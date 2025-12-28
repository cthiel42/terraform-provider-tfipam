package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccPoolDataSource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDataSourceConfig("test-pool", []string{"10.0.0.0/16"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-pool"),
					),
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/16"),
						}),
					),
				},
			},
		},
	})
}

func TestAccPoolDataSource_MultipleCIDRs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDataSourceConfig("multi-cidr-pool", []string{
					"10.0.0.0/16",
					"192.168.1.0/24",
					"172.16.0.0/12",
				}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("multi-cidr-pool"),
					),
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/16"),
							knownvalue.StringExact("192.168.1.0/24"),
							knownvalue.StringExact("172.16.0.0/12"),
						}),
					),
				},
			},
		},
	})
}

func TestAccPoolDataSource_IPv6(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDataSourceConfig("ipv6-pool", []string{"2001:db8::/32"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("ipv6-pool"),
					),
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("2001:db8::/32"),
						}),
					),
				},
			},
		},
	})
}

func TestAccPoolDataSource_MixedIPv4IPv6(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDataSourceConfig("mixed-pool", []string{
					"10.0.0.0/16",
					"2001:db8::/32",
					"192.168.1.0/24",
				}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("mixed-pool"),
					),
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/16"),
							knownvalue.StringExact("2001:db8::/32"),
							knownvalue.StringExact("192.168.1.0/24"),
						}),
					),
				},
			},
		},
	})
}

func TestAccPoolDataSource_NotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPoolDataSourceConfigNotFound("nonexistent-pool"),
				ExpectError: regexp.MustCompile("Provider produced null object|not found|does not exist"),
			},
		},
	})
}

func TestAccPoolDataSource_WithAllocations(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDataSourceConfigWithAllocations("pool-with-alloc", []string{"10.0.0.0/16"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("pool-with-alloc"),
					),
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/16"),
						}),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test1",
						tfjsonpath.New("pool_name"),
						knownvalue.StringExact("pool-with-alloc"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test2",
						tfjsonpath.New("pool_name"),
						knownvalue.StringExact("pool-with-alloc"),
					),
				},
			},
		},
	})
}

func TestAccPoolDataSource_UpdateResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create pool with one CIDR and read via data source
			{
				Config: testAccPoolDataSourceConfig("update-pool", []string{"10.0.0.0/16"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/16"),
						}),
					),
				},
			},
			// Update pool to add CIDRs and verify data source reflects change
			{
				Config: testAccPoolDataSourceConfig("update-pool", []string{
					"10.0.0.0/16",
					"192.168.1.0/24",
				}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/16"),
							knownvalue.StringExact("192.168.1.0/24"),
						}),
					),
				},
			},
		},
	})
}

func TestAccPoolDataSource_MultipleDataSources(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDataSourceConfigMultiple(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test1",
						tfjsonpath.New("name"),
						knownvalue.StringExact("pool-1"),
					),
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test2",
						tfjsonpath.New("name"),
						knownvalue.StringExact("pool-2"),
					),
					statecheck.ExpectKnownValue(
						"data.tfipam_pool.test3",
						tfjsonpath.New("name"),
						knownvalue.StringExact("pool-3"),
					),
				},
			},
		},
	})
}

// testAccPoolDataSourceConfig generates a Terraform configuration with a pool resource and data source.
func testAccPoolDataSourceConfig(name string, cidrs []string) string {
	cidrsConfig := ""
	for _, cidr := range cidrs {
		cidrsConfig += fmt.Sprintf("    %q,\n", cidr)
	}

	return fmt.Sprintf(`
resource "tfipam_pool" "test" {
  name = %[1]q
  cidrs = [
%[2]s  ]
}

data "tfipam_pool" "test" {
  name = tfipam_pool.test.name
}
`, name, cidrsConfig)
}

// testAccPoolDataSourceConfigNotFound generates a config that tries to read a non-existent pool.
func testAccPoolDataSourceConfigNotFound(name string) string {
	return fmt.Sprintf(`
data "tfipam_pool" "test" {
  name = %[1]q
}
`, name)
}

// testAccPoolDataSourceConfigWithAllocations generates a config with pool, allocations, and data source.
func testAccPoolDataSourceConfigWithAllocations(name string, cidrs []string) string {
	cidrsConfig := ""
	for _, cidr := range cidrs {
		cidrsConfig += fmt.Sprintf("    %q,\n", cidr)
	}

	return fmt.Sprintf(`
resource "tfipam_pool" "test" {
  name = %[1]q
  cidrs = [
%[2]s  ]
}

resource "tfipam_allocation" "test1" {
  id            = "alloc-1"
  pool_name     = tfipam_pool.test.name
  prefix_length = 24
}

resource "tfipam_allocation" "test2" {
  id            = "alloc-2"
  pool_name     = tfipam_pool.test.name
  prefix_length = 27
}

data "tfipam_pool" "test" {
  name = tfipam_pool.test.name
}
`, name, cidrsConfig)
}

// testAccPoolDataSourceConfigMultiple generates a config with multiple pools and data sources.
func testAccPoolDataSourceConfigMultiple() string {
	return `
resource "tfipam_pool" "pool1" {
  name = "pool-1"
  cidrs = ["10.0.0.0/16"]
}

resource "tfipam_pool" "pool2" {
  name = "pool-2"
  cidrs = ["192.168.0.0/16"]
}

resource "tfipam_pool" "pool3" {
  name = "pool-3"
  cidrs = ["172.16.0.0/12"]
}

data "tfipam_pool" "test1" {
  name = tfipam_pool.pool1.name
}

data "tfipam_pool" "test2" {
  name = tfipam_pool.pool2.name
}

data "tfipam_pool" "test3" {
  name = tfipam_pool.pool3.name
}
`
}
