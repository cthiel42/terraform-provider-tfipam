package provider

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &PoolResource{}
var _ resource.ResourceWithImportState = &PoolResource{}

func NewPoolResource() resource.Resource {
	return &PoolResource{}
}

// resource implementation
type PoolResource struct {
	provider *IpamProvider
}

// resource data model
type PoolResourceModel struct {
	Name        types.String `tfsdk:"name"`
	CIDRs       types.List   `tfsdk:"cidrs"`
	Allocations types.Map    `tfsdk:"allocations"`
}

func (r *PoolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool"
}

func (r *PoolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IPAM pool resource for managing IP address ranges",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the IP pool",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidrs": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "List of CIDR blocks in the pool",
			},
			"allocations": schema.MapAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "Map of allocation IDs to allocated IP addresses",
			},
		},
	}
}

func (r *PoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PoolResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate CIDRs
	var cidrs []string
	resp.Diagnostics.Append(data.CIDRs.ElementsAs(ctx, &cidrs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, cidr := range cidrs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			resp.Diagnostics.AddError(
				"Invalid CIDR",
				fmt.Sprintf("CIDR '%s' is not valid: %s", cidr, err),
			)
			return
		}
	}

	tflog.Trace(ctx, "created pool resource", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// init empty allocations map
	allocations := make(map[string]string)
	allocationsMap, diag := types.MapValueFrom(ctx, types.StringType, allocations)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Allocations = allocationsMap

	// register pool with provider
	if r.provider != nil {
		r.provider.mu.Lock()
		r.provider.pools[data.Name.ValueString()] = Pool{
			Name:  data.Name.ValueString(),
			CIDRs: cidrs,
		}
		r.provider.mu.Unlock()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PoolResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build allocations map from provider's runtime state
	if r.provider != nil {
		r.provider.mu.RLock()
		allocations := make(map[string]string)
		for id, alloc := range r.provider.allocations {
			if alloc.PoolName == data.Name.ValueString() {
				allocations[id] = alloc.AllocatedIP
			}
		}
		r.provider.mu.RUnlock()

		allocationsMap, diag := types.MapValueFrom(ctx, types.StringType, allocations)
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Allocations = allocationsMap
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PoolResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate CIDRs
	// TODO: Check for an allocations that would be invalidated by CIDR changes
	var cidrs []string
	resp.Diagnostics.Append(data.CIDRs.ElementsAs(ctx, &cidrs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, cidr := range cidrs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			resp.Diagnostics.AddError(
				"Invalid CIDR",
				fmt.Sprintf("CIDR '%s' is not valid: %s", cidr, err),
			)
			return
		}
	}

	tflog.Trace(ctx, "updated pool resource", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// Update pool registration with provider
	if r.provider != nil {
		r.provider.mu.Lock()
		r.provider.pools[data.Name.ValueString()] = Pool{
			Name:  data.Name.ValueString(),
			CIDRs: cidrs,
		}
		r.provider.mu.Unlock()
	}

	// Preserve allocations map from current state
	var currentState PoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentState)...)
	if !resp.Diagnostics.HasError() {
		data.Allocations = currentState.Allocations
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PoolResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// check for active allocations
	if r.provider != nil {
		r.provider.mu.RLock()
		hasAllocations := false
		for _, alloc := range r.provider.allocations {
			if alloc.PoolName == data.Name.ValueString() {
				hasAllocations = true
				break
			}
		}
		r.provider.mu.RUnlock()

		if hasAllocations {
			resp.Diagnostics.AddError(
				"Cannot Delete Pool",
				fmt.Sprintf("Pool %s has active allocations. Please delete all allocations before deleting the pool.", data.Name.ValueString()),
			)
			return
		}

		// remove pool from provider
		r.provider.mu.Lock()
		delete(r.provider.pools, data.Name.ValueString())
		r.provider.mu.Unlock()
	}

	tflog.Trace(ctx, "deleted pool resource", map[string]interface{}{
		"name": data.Name.ValueString(),
	})
}

func (r *PoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// import format: name:cidr1,cidr2,cidr3
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format: name:cidr1,cidr2,cidr3",
		)
		return
	}

	name := parts[0]
	cidrList := strings.Split(parts[1], ",")

	for _, cidr := range cidrList {
		if _, _, err := net.ParseCIDR(strings.TrimSpace(cidr)); err != nil {
			resp.Diagnostics.AddError(
				"Invalid CIDR",
				fmt.Sprintf("CIDR '%s' is not valid: %s", cidr, err),
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)

	// convert to terraform types so it can be set in diagnostics
	cidrElements := make([]types.String, len(cidrList))
	for i, cidr := range cidrList {
		cidrElements[i] = types.StringValue(strings.TrimSpace(cidr))
	}
	cidrsList, diag := types.ListValueFrom(ctx, types.StringType, cidrElements)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cidrs"), cidrsList)...)
}
