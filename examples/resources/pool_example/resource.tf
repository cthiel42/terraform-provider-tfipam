terraform {
  required_providers {
    tfipam = {
      source  = "cthiel42/tfipam"
      version = "1.0.3"
    }
  }
}

provider "tfipam" {}

resource "tfipam_pool" "example" {
  name = "pool_example"
  cidrs = [
    "10.0.0.0/24",
    "10.5.0.0/24"
  ]
}
