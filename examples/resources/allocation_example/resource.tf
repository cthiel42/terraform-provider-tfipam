terraform {
  required_providers {
    ipam = {
      source = "hashicorp.com/edu/tf-ipam"
    }
  }
}

provider "ipam" {}

resource "ipam_pool" "example" {
  name = "pool_example"
  cidrs = [
    "10.0.0.0/16",
    "10.5.0.0/24"
  ]
}

resource "ipam_allocation" "example" {
  pool_name     = ipam_pool.example.name
  prefix_length = 24
}

# resource "ipam_allocation" "example_2" {
#   pool_name     = ipam_pool.example.name
#   prefix_length = 24
# }
