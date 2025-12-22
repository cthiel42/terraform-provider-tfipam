package provider

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &AllocationResource{}
var _ resource.ResourceWithImportState = &AllocationResource{}

func NewAllocationResource() resource.Resource {
	return &AllocationResource{}
}

// resource implementation.
type AllocationResource struct {
	provider *IpamProvider
}

// resource data model.
type AllocationResourceModel struct {
	ID           types.String `tfsdk:"id"`
	PoolName     types.String `tfsdk:"pool_name"`
	AllocatedIP  types.String `tfsdk:"allocated_ip"`
	PrefixLength types.Int64  `tfsdk:"prefix_length"`
}

func (r *AllocationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_allocation"
}

func (r *AllocationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IPAM allocation resource for allocating IP addresses from a pool",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for this allocation",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pool_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the pool to allocate from",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allocated_ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The allocated IP address in CIDR notation",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"prefix_length": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Prefix length for the allocated IP (e.g., 32 for a single host)",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *AllocationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	provider, ok := req.ProviderData.(*IpamProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *IpamProvider, got: %T", req.ProviderData),
		)
		return
	}

	r.provider = provider
}

func (r *AllocationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AllocationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// validate prefix (0-128 for ipv6 compatibility)
	prefixLength := int(data.PrefixLength.ValueInt64())
	if prefixLength < 0 || prefixLength > 128 {
		resp.Diagnostics.AddError(
			"Invalid Prefix Length",
			fmt.Sprintf("Prefix length must be between 0 and 128, got %d", prefixLength),
		)
		return
	}

	// Find the pool and allocate the range
	poolName := data.PoolName.ValueString()
	allocatedIP, allocationID, err := r.allocateCIDRFromPool(ctx, poolName, prefixLength)
	if err != nil {
		resp.Diagnostics.AddError(
			"Allocation Failed",
			fmt.Sprintf("Unable to allocate CIDR from pool %s: %s", poolName, err),
		)
		return
	}

	data.ID = types.StringValue(allocationID)
	data.AllocatedIP = types.StringValue(allocatedIP)

	tflog.Trace(ctx, "created allocation resource", map[string]any{
		"id":           allocationID,
		"pool_name":    poolName,
		"allocated_ip": allocatedIP,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AllocationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AllocationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Register/update allocation in provider state during refresh
	if r.provider != nil {
		r.provider.mu.Lock()
		r.provider.allocations[data.ID.ValueString()] = Allocation{
			ID:           data.ID.ValueString(),
			PoolName:     data.PoolName.ValueString(),
			AllocatedIP:  data.AllocatedIP.ValueString(),
			PrefixLength: int(data.PrefixLength.ValueInt64()),
		}
		r.provider.mu.Unlock()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AllocationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes require replacement, so this should never be called
	var data AllocationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AllocationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AllocationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Remove allocation from provider state
	if r.provider != nil {
		r.provider.mu.Lock()
		delete(r.provider.allocations, data.ID.ValueString())
		r.provider.mu.Unlock()
	}

	tflog.Trace(ctx, "deleted allocation resource", map[string]any{
		"id":        data.ID.ValueString(),
		"pool_name": data.PoolName.ValueString(),
	})
}

func (r *AllocationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// For import we expect the ID to be the allocation ID
	// Need to look it up in the provider state
	allocationID := req.ID

	if r.provider == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"Provider is not configured for import",
		)
		return
	}

	r.provider.mu.RLock()
	allocation, exists := r.provider.allocations[allocationID]
	r.provider.mu.RUnlock()

	if !exists {
		resp.Diagnostics.AddError(
			"Allocation Not Found",
			fmt.Sprintf("Allocation %s not found in provider state", allocationID),
		)
		return
	}

	data := AllocationResourceModel{
		ID:           types.StringValue(allocation.ID),
		PoolName:     types.StringValue(allocation.PoolName),
		AllocatedIP:  types.StringValue(allocation.AllocatedIP),
		PrefixLength: types.Int64Value(int64(allocation.PrefixLength)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// allocateCIDRFromPool finds an available CIDR block in the pool and registers it.
// This implements a greedy search to find non-overlapping CIDR blocks
// of the requested size within the pool's CIDR ranges.
func (r *AllocationResource) allocateCIDRFromPool(ctx context.Context, poolName string, prefixLength int) (string, string, error) {
	if r.provider == nil {
		return "", "", fmt.Errorf("provider not configured")
	}

	r.provider.mu.Lock()
	defer r.provider.mu.Unlock()

	// get pool info
	pool, exists := r.provider.pools[poolName]
	if !exists {
		return "", "", fmt.Errorf("pool %s not found", poolName)
	}

	// collect all currently allocated CIDRs for this pool
	var allocatedCIDRs []*net.IPNet
	for _, alloc := range r.provider.allocations {
		if alloc.PoolName == poolName {
			_, allocNet, err := net.ParseCIDR(alloc.AllocatedIP)
			if err != nil {
				continue
			}
			allocatedCIDRs = append(allocatedCIDRs, allocNet)
		}
	}

	// find available CIDR block from each pool CIDR
	for _, poolCIDRStr := range pool.CIDRs {
		_, poolNet, err := net.ParseCIDR(poolCIDRStr)
		if err != nil {
			continue
		}

		poolPrefixLen, _ := poolNet.Mask.Size()

		// cant allocate a larger block than the pool itself
		if prefixLength < poolPrefixLen {
			continue 
		}

		// search for available cidr
		candidateCIDR := findAvailableCIDR(poolNet, prefixLength, allocatedCIDRs)
		if candidateCIDR != nil {
			allocatedIP := candidateCIDR.String()
			allocationID := fmt.Sprintf("%s-%s", poolName, allocatedIP)

			// Register the allocation in provider state
			r.provider.allocations[allocationID] = Allocation{
				ID:           allocationID,
				PoolName:     poolName,
				AllocatedIP:  allocatedIP,
				PrefixLength: prefixLength,
			}

			return allocatedIP, allocationID, nil
		}
	}

	return "", "", fmt.Errorf("no available CIDR blocks of size /%d in pool %s", prefixLength, poolName)
}

// findAvailableCIDR searches for an available CIDR block of the requested prefix length
// within the pool CIDR such that it doesn't overlap with any existing allocations
func findAvailableCIDR(poolNet *net.IPNet, prefixLength int, allocatedCIDRs []*net.IPNet) *net.IPNet {
	poolPrefixLen, bits := poolNet.Mask.Size()

	// Calculate number of blocks of the requested size that can fit in the pool
	blockSizeDiff := prefixLength - poolPrefixLen
	if blockSizeDiff < 0 {
		return nil // Requested block is larger than pool
	}
	numBlocks := 1 << uint(blockSizeDiff) // 2^(prefixLength - poolPrefixLen)

	requestedMask := net.CIDRMask(prefixLength, bits)

	// Iterate through all possible CIDR blocks of the requested size within the pool
	// and check if they overlap with existing allocations
	baseIP := poolNet.IP
	for i := 0; i < numBlocks; i++ {
		candidateIP := make(net.IP, len(baseIP))
		copy(candidateIP, baseIP)
		addIPOffset(candidateIP, i, prefixLength, bits)
		candidateNet := &net.IPNet{
			IP:   candidateIP.Mask(requestedMask),
			Mask: requestedMask,
		}

		// edge cases. ensure IP is in pool and last ip is in pool
		if !poolNet.Contains(candidateNet.IP) {
			continue
		}
		lastIP := getLastIPInCIDR(candidateNet)
		if !poolNet.Contains(lastIP) {
			continue
		}

		// check for overlaps with existing allocations
		if !cidrsOverlap(candidateNet, allocatedCIDRs) {
			return candidateNet
		}
	}

	return nil
}

// addIPOffset adds an offset to an IP address based on block size
func addIPOffset(ip net.IP, blockIndex int, prefixLength int, totalBits int) {
	// calculate IPs per block
	hostBits := totalBits - prefixLength
	blockSize := 1 << uint(hostBits)
	offset := blockIndex * blockSize

	// Add the offset to the IP address (big-endian)
	if len(ip) == 4 {
		// IPv4
		ipInt := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
		ipInt += uint32(offset)
		ip[0] = byte(ipInt >> 24)
		ip[1] = byte(ipInt >> 16)
		ip[2] = byte(ipInt >> 8)
		ip[3] = byte(ipInt)
	} else {
		// IPv6 - add offset from the right
		for i := len(ip) - 1; i >= 0 && offset > 0; i-- {
			offset += int(ip[i])
			ip[i] = byte(offset & 0xFF)
			offset >>= 8
		}
	}
}

// getLastIPInCIDR returns the last IP address in a CIDR block
func getLastIPInCIDR(cidr *net.IPNet) net.IP {
	ip := make(net.IP, len(cidr.IP))
	copy(ip, cidr.IP)

	// invert the mask and OR it with the IP to get the last address
	for i := range ip {
		ip[i] |= ^cidr.Mask[i]
	}

	return ip
}

// cidrsOverlap checks if a candidate CIDR overlaps with any allocated CIDRs
func cidrsOverlap(candidate *net.IPNet, allocated []*net.IPNet) bool {
	for _, allocNet := range allocated {
		// Check if either CIDR contains the other's network address
		if candidate.Contains(allocNet.IP) || allocNet.Contains(candidate.IP) {
			return true
		}

		// Also check if the last IP of candidate is in allocated or vice versa
		candidateLastIP := getLastIPInCIDR(candidate)
		allocLastIP := getLastIPInCIDR(allocNet)

		if candidate.Contains(allocLastIP) || allocNet.Contains(candidateLastIP) {
			return true
		}
	}

	return false
}
