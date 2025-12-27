terraform {
  required_providers {
    tfipam = {
      source  = "cthiel42/tfipam"
      version = "1.1.0"
    }
  }
}

provider "tfipam" {}

resource "tfipam_pool" "example" {
  name = "pool_example"
  cidrs = [
    "10.0.0.0/16",
    "10.5.0.0/24"
  ]
}

resource "tfipam_allocation" "example_0" {
  id            = "allocation_example_0"
  pool_name     = tfipam_pool.example.name
  prefix_length = 24
}

resource "tfipam_allocation" "example_1" {
  id            = "allocation_example_1"
  pool_name     = tfipam_pool.example.name
  prefix_length = 27
}