package provider

import (
	"context"
	"fmt"
	"terraform-provider-tfipam/internal/provider/storage"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &PoolDataSource{}

func NewPoolDataSource() datasource.DataSource {
	return &PoolDataSource{}
}

type PoolDataSource struct {
	provider *IpamProvider
}

type PoolDataSourceModel struct {
	Name  types.String `tfsdk:"name"`
	CIDRs types.List   `tfsdk:"cidrs"`
}

func (d *PoolDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool"
}

func (d *PoolDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IPAM pool data source for managing IP address ranges",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the IP pool",
				Required:            true,
			},
			"cidrs": schema.ListAttribute{
				MarkdownDescription: "CIDR blocks in the pool",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *PoolDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PoolDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pool, err := d.provider.storage.GetPool(ctx, data.Name.ValueString())
	if err != nil {
		// handle not found error by removing resource from state
		if err == storage.ErrNotFound {
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
