package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"tfipam": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// You can add precondition checks here
	// For example, checking environment variables or other prerequisites
}

func TestAccPoolResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPoolResourceConfig("test-pool", []string{"10.0.0.0/16"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-pool"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/16"),
						}),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:                         "tfipam_pool.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "test-pool:10.0.0.0/16",
				ImportStateVerifyIdentifierAttribute: "name",
			},
			// Update and Read testing
			{
				Config: testAccPoolResourceConfig("test-pool", []string{"10.0.0.0/16", "192.168.1.0/24"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-pool"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/16"),
							knownvalue.StringExact("192.168.1.0/24"),
						}),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccPoolResource_MultipleCIDRs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with multiple CIDRs
			{
				Config: testAccPoolResourceConfig("multi-cidr-pool", []string{
					"10.0.0.0/16",
					"192.168.1.0/24",
					"172.16.0.0/12",
				}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("multi-cidr-pool"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
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

func TestAccPoolResource_UpdateCIDRs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with one CIDR
			{
				Config: testAccPoolResourceConfig("update-pool", []string{"10.0.0.0/16"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/16"),
						}),
					),
				},
			},
			// Add a CIDR
			{
				Config: testAccPoolResourceConfig("update-pool", []string{
					"10.0.0.0/16",
					"192.168.0.0/16",
				}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/16"),
							knownvalue.StringExact("192.168.0.0/16"),
						}),
					),
				},
			},
			// Remove a CIDR
			{
				Config: testAccPoolResourceConfig("update-pool", []string{"192.168.0.0/16"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("cidrs"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("192.168.0.0/16"),
						}),
					),
				},
			},
		},
	})
}

func TestAccPoolResource_InvalidCIDR(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPoolResourceConfig("invalid-pool", []string{"not-a-valid-cidr"}),
				ExpectError: regexp.MustCompile("Invalid CIDR"),
			},
		},
	})
}

func TestAccPoolResource_NameChange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with original name
			{
				Config: testAccPoolResourceConfig("original-name", []string{"10.0.0.0/16"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("original-name"),
					),
				},
			},
			// Change name (should trigger replacement due to RequiresReplace modifier)
			{
				Config: testAccPoolResourceConfig("new-name", []string{"10.0.0.0/16"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("new-name"),
					),
				},
			},
		},
	})
}

func TestAccPoolResource_ImportBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create initial resource
			{
				Config: testAccPoolResourceConfig("import-test", []string{"10.0.0.0/16"}),
			},
			// ImportState testing
			{
				ResourceName:                         "tfipam_pool.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "import-test:10.0.0.0/16",
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccPoolResource_ImportMultipleCIDRs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create initial resource with multiple CIDRs
			{
				Config: testAccPoolResourceConfig("import-multi", []string{
					"10.0.0.0/16",
					"192.168.1.0/24",
					"172.16.0.0/12",
				}),
			},
			// ImportState testing with multiple CIDRs
			{
				ResourceName:                         "tfipam_pool.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "import-multi:10.0.0.0/16,192.168.1.0/24,172.16.0.0/12",
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccPoolResource_ImportWithSpaces(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create initial resource
			{
				Config: testAccPoolResourceConfig("import-spaces", []string{
					"10.0.0.0/16",
					"192.168.1.0/24",
				}),
			},
			// ImportState testing with spaces in import ID (should handle gracefully)
			{
				ResourceName:                         "tfipam_pool.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "import-spaces: 10.0.0.0/16 , 192.168.1.0/24 ",
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccPoolResource_WithAllocations(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create pool and allocation
			{
				Config: testAccPoolResourceConfigWithAllocation("pool-with-alloc", []string{"10.0.0.0/16"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("pool-with-alloc"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("pool_name"),
						knownvalue.StringExact("pool-with-alloc"),
					),
				},
			},
		},
	})
}

func TestAccPoolResource_IPv6(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolResourceConfig("ipv6-pool", []string{"2001:db8::/32"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("ipv6-pool"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
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

func TestAccPoolResource_MixedIPv4IPv6(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolResourceConfig("mixed-pool", []string{
					"10.0.0.0/16",
					"2001:db8::/32",
					"192.168.1.0/24",
				}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("mixed-pool"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_pool.test",
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

// testAccPoolResourceConfig generates a Terraform configuration for a pool resource.
func testAccPoolResourceConfig(name string, cidrs []string) string {
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
`, name, cidrsConfig)
}

// testAccPoolResourceConfigWithAllocation generates a Terraform configuration for a pool resource with an allocation.
func testAccPoolResourceConfigWithAllocation(name string, cidrs []string) string {
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

resource "tfipam_allocation" "test" {
  id            = "test-allocation"
  pool_name     = tfipam_pool.test.name
  prefix_length = 24
}
`, name, cidrsConfig)
}
