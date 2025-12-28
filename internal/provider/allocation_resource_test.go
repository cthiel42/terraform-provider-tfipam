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

func TestAccAllocationResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccAllocationResourceConfig("test-pool", "test-alloc", 24),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("test-alloc"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("pool_name"),
						knownvalue.StringExact("test-pool"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(24),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("allocated_cidr"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "tfipam_allocation.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "test-alloc",
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccAllocationResource_MultipleAllocations(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create pool and multiple allocations
			{
				Config: testAccAllocationResourceConfigMultiple("multi-pool", 24),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test1",
						tfjsonpath.New("id"),
						knownvalue.StringExact("alloc-1"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test2",
						tfjsonpath.New("id"),
						knownvalue.StringExact("alloc-2"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test3",
						tfjsonpath.New("id"),
						knownvalue.StringExact("alloc-3"),
					),
					// Verify all have allocated CIDRs
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test1",
						tfjsonpath.New("allocated_cidr"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test2",
						tfjsonpath.New("allocated_cidr"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test3",
						tfjsonpath.New("allocated_cidr"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccAllocationResource_DifferentPrefixLengths(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAllocationResourceConfigDifferentPrefixes("prefix-pool"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test_24",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(24),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test_27",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(27),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test_30",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(30),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test_32",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(32),
					),
				},
			},
		},
	})
}

func TestAccAllocationResource_SingleHost(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAllocationResourceConfig("host-pool", "host-alloc", 32),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(32),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("allocated_cidr"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccAllocationResource_InvalidPrefixLength_Negative(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAllocationResourceConfig("invalid-pool", "invalid-alloc", -1),
				ExpectError: regexp.MustCompile("Invalid Prefix Length"),
			},
		},
	})
}

func TestAccAllocationResource_InvalidPrefixLength_TooLarge(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAllocationResourceConfig("invalid-pool", "invalid-alloc", 129),
				ExpectError: regexp.MustCompile("Invalid Prefix Length"),
			},
		},
	})
}

func TestAccAllocationResource_PrefixLargerThanPool(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAllocationResourceConfigSmallPool("small-pool", "too-large", 16),
				ExpectError: regexp.MustCompile("no available CIDR blocks|Allocation Failed"),
			},
		},
	})
}

func TestAccAllocationResource_PoolNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAllocationResourceConfigNoPool("nonexistent-pool", "test-alloc", 24),
				ExpectError: regexp.MustCompile("pool.*not found|Allocation Failed"),
			},
		},
	})
}

func TestAccAllocationResource_IDChange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with original ID
			{
				Config: testAccAllocationResourceConfig("id-pool", "original-id", 24),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("original-id"),
					),
				},
			},
			// Change ID (should trigger replacement)
			{
				Config: testAccAllocationResourceConfig("id-pool", "new-id", 24),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("new-id"),
					),
				},
			},
		},
	})
}

func TestAccAllocationResource_PoolChange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with original pool
			{
				Config: testAccAllocationResourceConfigTwoPools("pool-1", "pool-2", "test-alloc", 24, "pool-1"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("pool_name"),
						knownvalue.StringExact("pool-1"),
					),
				},
			},
			// Change pool (should trigger replacement)
			{
				Config: testAccAllocationResourceConfigTwoPools("pool-1", "pool-2", "test-alloc", 24, "pool-2"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("pool_name"),
						knownvalue.StringExact("pool-2"),
					),
				},
			},
		},
	})
}

func TestAccAllocationResource_PrefixLengthChange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with /24
			{
				Config: testAccAllocationResourceConfig("prefix-pool", "prefix-alloc", 24),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(24),
					),
				},
			},
			// Change to /27 (should trigger replacement)
			{
				Config: testAccAllocationResourceConfig("prefix-pool", "prefix-alloc", 27),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(27),
					),
				},
			},
		},
	})
}

func TestAccAllocationResource_Import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create allocation
			{
				Config: testAccAllocationResourceConfig("import-pool", "import-alloc", 24),
			},
			// Import by ID
			{
				ResourceName:      "tfipam_allocation.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "import-alloc",
			},
		},
	})
}

func TestAccAllocationResource_IPv6(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAllocationResourceConfigIPv6("ipv6-pool", "ipv6-alloc", 64),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("ipv6-alloc"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(64),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test",
						tfjsonpath.New("allocated_cidr"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccAllocationResource_IPv6_MultipleSubnets(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAllocationResourceConfigIPv6Multiple("ipv6-multi-pool"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test_48",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(48),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test_56",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(56),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test_64",
						tfjsonpath.New("prefix_length"),
						knownvalue.Int64Exact(64),
					),
				},
			},
		},
	})
}

func TestAccAllocationResource_SequentialAllocations(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Start with one allocation
			{
				Config: testAccAllocationResourceConfigSequential("seq-pool", 1, 27),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test_0",
						tfjsonpath.New("id"),
						knownvalue.StringExact("seq-alloc-0"),
					),
				},
			},
			// Add second allocation
			{
				Config: testAccAllocationResourceConfigSequential("seq-pool", 2, 27),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test_0",
						tfjsonpath.New("id"),
						knownvalue.StringExact("seq-alloc-0"),
					),
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test_1",
						tfjsonpath.New("id"),
						knownvalue.StringExact("seq-alloc-1"),
					),
				},
			},
			// Add third allocation
			{
				Config: testAccAllocationResourceConfigSequential("seq-pool", 3, 27),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"tfipam_allocation.test_2",
						tfjsonpath.New("id"),
						knownvalue.StringExact("seq-alloc-2"),
					),
				},
			},
		},
	})
}

// testAccAllocationResourceConfig generates a Terraform configuration for an allocation resource.
func testAccAllocationResourceConfig(poolName, allocID string, prefixLength int) string {
	return fmt.Sprintf(`
resource "tfipam_pool" "test" {
  name = %[1]q
  cidrs = ["10.0.0.0/16"]
}

resource "tfipam_allocation" "test" {
  id            = %[2]q
  pool_name     = tfipam_pool.test.name
  prefix_length = %[3]d
}
`, poolName, allocID, prefixLength)
}

// testAccAllocationResourceConfigNoPool generates config without creating the pool first.
func testAccAllocationResourceConfigNoPool(poolName, allocID string, prefixLength int) string {
	return fmt.Sprintf(`
resource "tfipam_allocation" "test" {
  id            = %[1]q
  pool_name     = %[2]q
  prefix_length = %[3]d
}
`, allocID, poolName, prefixLength)
}

// testAccAllocationResourceConfigMultiple generates config with multiple allocations.
func testAccAllocationResourceConfigMultiple(poolName string, prefixLength int) string {
	return fmt.Sprintf(`
resource "tfipam_pool" "test" {
  name = %[1]q
  cidrs = ["10.0.0.0/16"]
}

resource "tfipam_allocation" "test1" {
  id            = "alloc-1"
  pool_name     = tfipam_pool.test.name
  prefix_length = %[2]d
}

resource "tfipam_allocation" "test2" {
  id            = "alloc-2"
  pool_name     = tfipam_pool.test.name
  prefix_length = %[2]d
}

resource "tfipam_allocation" "test3" {
  id            = "alloc-3"
  pool_name     = tfipam_pool.test.name
  prefix_length = %[2]d
}
`, poolName, prefixLength)
}

// testAccAllocationResourceConfigDifferentPrefixes generates config with different prefix lengths.
func testAccAllocationResourceConfigDifferentPrefixes(poolName string) string {
	return fmt.Sprintf(`
resource "tfipam_pool" "test" {
  name = %[1]q
  cidrs = ["10.0.0.0/16"]
}

resource "tfipam_allocation" "test_24" {
  id            = "alloc-24"
  pool_name     = tfipam_pool.test.name
  prefix_length = 24
}

resource "tfipam_allocation" "test_27" {
  id            = "alloc-27"
  pool_name     = tfipam_pool.test.name
  prefix_length = 27
}

resource "tfipam_allocation" "test_30" {
  id            = "alloc-30"
  pool_name     = tfipam_pool.test.name
  prefix_length = 30
}

resource "tfipam_allocation" "test_32" {
  id            = "alloc-32"
  pool_name     = tfipam_pool.test.name
  prefix_length = 32
}
`, poolName)
}

// testAccAllocationResourceConfigSmallPool generates config with a small pool.
func testAccAllocationResourceConfigSmallPool(poolName, allocID string, prefixLength int) string {
	return fmt.Sprintf(`
resource "tfipam_pool" "test" {
  name = %[1]q
  cidrs = ["10.0.0.0/24"]
}

resource "tfipam_allocation" "test" {
  id            = %[2]q
  pool_name     = tfipam_pool.test.name
  prefix_length = %[3]d
}
`, poolName, allocID, prefixLength)
}

// testAccAllocationResourceConfigTwoPools generates config with two pools.
func testAccAllocationResourceConfigTwoPools(pool1, pool2, allocID string, prefixLength int, usePool string) string {
	return fmt.Sprintf(`
resource "tfipam_pool" "pool1" {
  name = %[1]q
  cidrs = ["10.0.0.0/16"]
}

resource "tfipam_pool" "pool2" {
  name = %[2]q
  cidrs = ["192.168.0.0/16"]
}

resource "tfipam_allocation" "test" {
  id            = %[3]q
  pool_name     = %[5]s
  prefix_length = %[4]d
}
`, pool1, pool2, allocID, prefixLength,
		func() string {
			if usePool == pool1 {
				return "tfipam_pool.pool1.name"
			}
			return "tfipam_pool.pool2.name"
		}())
}

// testAccAllocationResourceConfigIPv6 generates config for IPv6 allocation.
func testAccAllocationResourceConfigIPv6(poolName, allocID string, prefixLength int) string {
	return fmt.Sprintf(`
resource "tfipam_pool" "test" {
  name = %[1]q
  cidrs = ["2001:db8::/32"]
}

resource "tfipam_allocation" "test" {
  id            = %[2]q
  pool_name     = tfipam_pool.test.name
  prefix_length = %[3]d
}
`, poolName, allocID, prefixLength)
}

// testAccAllocationResourceConfigIPv6Multiple generates config for multiple IPv6 allocations.
func testAccAllocationResourceConfigIPv6Multiple(poolName string) string {
	return fmt.Sprintf(`
resource "tfipam_pool" "test" {
  name = %[1]q
  cidrs = ["2001:db8::/32"]
}

resource "tfipam_allocation" "test_48" {
  id            = "ipv6-alloc-48"
  pool_name     = tfipam_pool.test.name
  prefix_length = 48
}

resource "tfipam_allocation" "test_56" {
  id            = "ipv6-alloc-56"
  pool_name     = tfipam_pool.test.name
  prefix_length = 56
}

resource "tfipam_allocation" "test_64" {
  id            = "ipv6-alloc-64"
  pool_name     = tfipam_pool.test.name
  prefix_length = 64
}
`, poolName)
}

// testAccAllocationResourceConfigSequential generates config with sequential allocations.
func testAccAllocationResourceConfigSequential(poolName string, count int, prefixLength int) string {
	config := fmt.Sprintf(`
resource "tfipam_pool" "test" {
  name = %[1]q
  cidrs = ["10.0.0.0/24"]
}
`, poolName)

	for i := 0; i < count; i++ {
		config += fmt.Sprintf(`
resource "tfipam_allocation" "test_%[1]d" {
  id            = "seq-alloc-%[1]d"
  pool_name     = tfipam_pool.test.name
  prefix_length = %[2]d
}
`, i, prefixLength)
	}

	return config
}
