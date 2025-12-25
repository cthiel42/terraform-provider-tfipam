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

	"terraform-provider-tfipam/internal/provider/storage"
)

var _ resource.Resource = &PoolResource{}
var _ resource.ResourceWithImportState = &PoolResource{}

func NewPoolResource() resource.Resource {
	return &PoolResource{}
}

type PoolResource struct {
	provider *IpamProvider
}

type PoolResourceModel struct {
	Name  types.String `tfsdk:"name"`
	CIDRs types.List   `tfsdk:"cidrs"`
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

	// validate cidrs
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

	// save pool to storage
	pool := &storage.Pool{
		Name:  data.Name.ValueString(),
		CIDRs: cidrs,
	}

	if err := r.provider.storage.SavePool(ctx, pool); err != nil {
		resp.Diagnostics.AddError(
			"Failed to Save Pool",
			fmt.Sprintf("Could not save pool to storage: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "created pool resource", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PoolResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pool, err := r.provider.storage.GetPool(ctx, data.Name.ValueString())
	if err != nil {
		if err == storage.ErrNotFound {
			// pool was deleted outside terraform, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Failed to Read Pool",
			fmt.Sprintf("Could not read pool from storage: %s", err),
		)
		return
	}

	// sync state with storage data
	cidrs, diag := types.ListValueFrom(ctx, types.StringType, pool.CIDRs)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.CIDRs = cidrs

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PoolResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// validate cidrs
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

	// TODO: Check for allocations that would be invalidated by CIDR changes to the pool

	// Update pool in storage
	pool := &storage.Pool{
		Name:  data.Name.ValueString(),
		CIDRs: cidrs,
	}

	if err := r.provider.storage.SavePool(ctx, pool); err != nil {
		resp.Diagnostics.AddError(
			"Failed to Update Pool",
			fmt.Sprintf("Could not update pool in storage: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "updated pool resource", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PoolResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolName := data.Name.ValueString()

	// check for active allocations in storage
	allocations, err := r.provider.storage.ListAllocationsByPool(ctx, poolName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Check Allocations",
			fmt.Sprintf("Could not check for allocations: %s", err),
		)
		return
	}

	if len(allocations) > 0 {
		resp.Diagnostics.AddError(
			"Cannot Delete Pool",
			fmt.Sprintf("Pool %s has %d active allocations. Please delete all allocations before deleting the pool.", poolName, len(allocations)),
		)
		return
	}

	err = r.provider.storage.DeletePool(ctx, poolName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Delete Pool",
			fmt.Sprintf("Could not delete pool from storage: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "deleted pool resource", map[string]interface{}{
		"name": poolName,
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

	// validate cidrs
	cidrs := make([]string, 0, len(cidrList))
	for _, cidr := range cidrList {
		trimmed := strings.TrimSpace(cidr)
		if _, _, err := net.ParseCIDR(trimmed); err != nil {
			resp.Diagnostics.AddError(
				"Invalid CIDR",
				fmt.Sprintf("CIDR '%s' is not valid: %s", cidr, err),
			)
			return
		}
		cidrs = append(cidrs, trimmed)
	}

	pool := &storage.Pool{
		Name:  name,
		CIDRs: cidrs,
	}

	if err := r.provider.storage.SavePool(ctx, pool); err != nil {
		resp.Diagnostics.AddError(
			"Failed to Import Pool",
			fmt.Sprintf("Could not save imported pool to storage: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
	cidrsList, diag := types.ListValueFrom(ctx, types.StringType, cidrs)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cidrs"), cidrsList)...)
}
