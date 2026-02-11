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

	"terraform-provider-tfipam/internal/provider/storage"
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
	StorageType           types.String `tfsdk:"storage_type"`
	FilePath              types.String `tfsdk:"file_path"`
	AzureConnectionString types.String `tfsdk:"azure_connection_string"`
	AzureContainerName    types.String `tfsdk:"azure_container_name"`
	AzureBlobName         types.String `tfsdk:"azure_blob_name"`
	S3Region              types.String `tfsdk:"s3_region"`
	S3BucketName          types.String `tfsdk:"s3_bucket_name"`
	S3ObjectKey           types.String `tfsdk:"s3_object_key"`
	S3AccessKeyID         types.String `tfsdk:"s3_access_key_id"`
	S3SecretAccessKey     types.String `tfsdk:"s3_secret_access_key"`
	S3SessionToken        types.String `tfsdk:"s3_session_token"`
	S3EndpointURL         types.String `tfsdk:"s3_endpoint_url"`
	S3SkipTLSVerify       types.Bool   `tfsdk:"s3_skip_tls_verify"`
}

func (p *IpamProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tfipam"
	resp.Version = p.version
}

func (p *IpamProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IPAM provider for managing IP address pools and allocations",
		Attributes: map[string]schema.Attribute{
			"storage_type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Storage backend type. Supported values: 'file' (default), 'azure_blob' (Azure Blob Storage), 'aws_s3' (AWS S3)",
			},
			"file_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Path to storage file for 'file' storage backend. Required for 'file' backend. Defaults to '.terraform/ipam-storage.json'",
			},
			"azure_connection_string": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Connection string for Azure Blob Storage. Required for 'azure_blob' backend.",
			},
			"azure_container_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Container name for Azure Blob Storage. Required for 'azure_blob' backend.",
			},
			"azure_blob_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Blob name for Azure Blob Storage. Defaults to 'ipam-storage.json'",
			},
			"s3_region": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "AWS region for S3 bucket. Required for 'aws_s3' backend.",
			},
			"s3_bucket_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "S3 bucket name. Required for 'aws_s3' backend.",
			},
			"s3_object_key": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "S3 object key (file path). Defaults to 'ipam-storage.json'",
			},
			"s3_access_key_id": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "AWS Access Key ID. Optional - uses default AWS credential chain if not provided.",
			},
			"s3_secret_access_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "AWS Secret Access Key. Required if s3_access_key_id is provided.",
			},
			"s3_session_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "AWS Session Token. Optional - for temporary credentials.",
			},
			"s3_endpoint_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Custom S3 endpoint URL. Optional - for S3 compatible services like MinIO or LocalStack.",
			},
			"s3_skip_tls_verify": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Skip TLS certificate verification. Optional - can be useful with self signed certificates on S3 compatible services",
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

		storageConfig := &storage.Config{
			Type: storageType,
		}

		// File backend config
		if !data.FilePath.IsNull() && !data.FilePath.IsUnknown() {
			storageConfig.FilePath = data.FilePath.ValueString()
		}

		// Azure backend config
		if !data.AzureConnectionString.IsNull() && !data.AzureConnectionString.IsUnknown() {
			storageConfig.AzureConnectionString = data.AzureConnectionString.ValueString()
		}
		if !data.AzureContainerName.IsNull() && !data.AzureContainerName.IsUnknown() {
			storageConfig.AzureContainerName = data.AzureContainerName.ValueString()
		}
		if !data.AzureBlobName.IsNull() && !data.AzureBlobName.IsUnknown() {
			storageConfig.AzureBlobName = data.AzureBlobName.ValueString()
		}

		// S3 backend config
		if !data.S3Region.IsNull() && !data.S3Region.IsUnknown() {
			storageConfig.S3Region = data.S3Region.ValueString()
		}
		if !data.S3BucketName.IsNull() && !data.S3BucketName.IsUnknown() {
			storageConfig.S3BucketName = data.S3BucketName.ValueString()
		}
		if !data.S3ObjectKey.IsNull() && !data.S3ObjectKey.IsUnknown() {
			storageConfig.S3ObjectKey = data.S3ObjectKey.ValueString()
		}
		if !data.S3AccessKeyID.IsNull() && !data.S3AccessKeyID.IsUnknown() {
			storageConfig.S3AccessKeyID = data.S3AccessKeyID.ValueString()
		}
		if !data.S3SecretAccessKey.IsNull() && !data.S3SecretAccessKey.IsUnknown() {
			storageConfig.S3SecretAccessKey = data.S3SecretAccessKey.ValueString()
		}
		if !data.S3SessionToken.IsNull() && !data.S3SessionToken.IsUnknown() {
			storageConfig.S3SessionToken = data.S3SessionToken.ValueString()
		}
		if !data.S3EndpointURL.IsNull() && !data.S3EndpointURL.IsUnknown() {
			storageConfig.S3EndpointURL = data.S3EndpointURL.ValueString()
		}
		if !data.S3SkipTLSVerify.IsNull() && !data.S3SkipTLSVerify.IsUnknown() {
			storageConfig.S3SkipTLSVerify = data.S3SkipTLSVerify.ValueBool()
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
			"type": storageConfig.Type,
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
