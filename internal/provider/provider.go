package provider

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

	// Runtime coordination state for resources
	mu          sync.RWMutex
	pools       map[string]Pool       // pool_name -> pool details
	allocations map[string]Allocation // allocation_id -> allocation details
}

// pool representation for runtime state
type Pool struct {
	Name  string
	CIDRs []string
}

// allocation representation for runtime state
type Allocation struct {
	ID           string
	PoolName     string
	AllocatedIP  string
	PrefixLength int
}

// provider data model.
type IpamProviderModel struct{}

func (p *IpamProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ipam"
	resp.Version = p.version
}

func (p *IpamProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{}
}

func (p *IpamProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data IpamProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// init coordination maps
	if p.pools == nil {
		p.pools = make(map[string]Pool)
	}
	if p.allocations == nil {
		p.allocations = make(map[string]Allocation)
	}
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
		NewExampleDataSource,
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
