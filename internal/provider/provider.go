package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-tf-ipam/internal/provider/storage"
)

var _ provider.Provider = &IpamProvider{}
var _ provider.ProviderWithFunctions = &IpamProvider{}
var _ provider.ProviderWithEphemeralResources = &IpamProvider{}
var _ provider.ProviderWithActions = &IpamProvider{}

type IpamProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string

	// storage backend for persistent state
	storage storage.Storage
}

// provider data model.
type IpamProviderModel struct {
	StorageType types.String `tfsdk:"storage_type"`
	FilePath    types.String `tfsdk:"file_path"`
}

func (p *IpamProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ipam"
	resp.Version = p.version
}

func (p *IpamProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IPAM provider for managing IP address pools and allocations",
		Attributes: map[string]schema.Attribute{
			"storage_type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Storage backend type. Supported values: 'file' (default)",
			},
			"file_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Path to storage file for 'file' storage backend. Defaults to '.terraform/ipam-storage.json'",
			},
		},
	}
}

func (p *IpamProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data IpamProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// set up storage backend
	if p.storage == nil {
		storageType := "file"
		if !data.StorageType.IsNull() && !data.StorageType.IsUnknown() {
			storageType = data.StorageType.ValueString()
		}

		filePath := ""
		if !data.FilePath.IsNull() && !data.FilePath.IsUnknown() {
			filePath = data.FilePath.ValueString()
		}

		storageConfig := &storage.Config{
			Type:     storageType,
			FilePath: filePath,
		}

		var err error
		p.storage, err = storage.Factory(ctx, storageConfig)
		if err != nil {
			resp.Diagnostics.AddError(
				"Storage Initialization Failed",
				fmt.Sprintf("Failed to initialize storage backend: %s", err),
			)
			return
		}

		tflog.Debug(ctx, "Storage backend initialized", map[string]any{
			"type":      storageConfig.Type,
			"file_path": storageConfig.FilePath,
		})
	}

	// Pass provider instance to resources so they can access storage
	resp.ResourceData = p
	resp.DataSourceData = p

	tflog.Debug(ctx, "Provider configured successfully", map[string]any{
		"provider_ptr": fmt.Sprintf("%p", p),
	})
}

func (p *IpamProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPoolResource,
		NewAllocationResource,
	}
}

func (p *IpamProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *IpamProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPoolDataSource,
		NewAllocationDataSource,
	}
}

func (p *IpamProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func (p *IpamProvider) Actions(ctx context.Context) []func() action.Action {
	return []func() action.Action{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &IpamProvider{
			version: version,
		}
	}
}
