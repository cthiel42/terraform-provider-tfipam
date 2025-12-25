package provider

import (
	"context"
	"fmt"
	"terraform-provider-tfipam/internal/provider/storage"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AllocationDataSource{}

func NewAllocationDataSource() datasource.DataSource {
	return &AllocationDataSource{}
}

type AllocationDataSource struct {
	provider *IpamProvider
}

type AllocationDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	PoolName      types.String `tfsdk:"pool_name"`
	AllocatedCIDR types.String `tfsdk:"allocated_cidr"`
	PrefixLength  types.Int64  `tfsdk:"prefix_length"`
}

func (d *AllocationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_allocation"
}

func (d *AllocationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Allocation data source for retrieving IP allocations from a pool",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the allocation",
				Required:            true,
			},
			"pool_name": schema.StringAttribute{
				MarkdownDescription: "Name of the pool the allocation belongs to",
				Required:            true,
			},
			"allocated_cidr": schema.StringAttribute{
				MarkdownDescription: "CIDR block allocated to the resource",
				Computed:            true,
			},
			"prefix_length": schema.Int64Attribute{
				MarkdownDescription: "Prefix length of the allocated CIDR",
				Computed:            true,
			},
		},
	}
}

func (d *AllocationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.provider = provider
}

func (d *AllocationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AllocationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allocation, err := d.provider.storage.GetAllocation(ctx, data.ID.ValueString())
	if err != nil {
		if err == storage.ErrNotFound {
			// allocation was deleted outside Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Failed to Read Allocation",
			fmt.Sprintf("Could not read allocation from storage: %s", err),
		)
		return
	}

	// sync state with storage data
	data.AllocatedCIDR = types.StringValue(allocation.AllocatedCIDR)
	data.PoolName = types.StringValue(allocation.PoolName)
	data.PrefixLength = types.Int64Value(int64(allocation.PrefixLength))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
