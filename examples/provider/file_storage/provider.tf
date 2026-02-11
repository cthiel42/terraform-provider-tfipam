terraform {
  required_providers {
    tfipam = {
      source  = "cthiel42/tfipam"
      version = "1.2.0"
    }
  }
}

provider "tfipam" {
  storage_type = "file"
  file_path    = "ipam_storage_example.json"
}

resource "tfipam_pool" "example" {
  name = "pool_example"
  cidrs = [
    "10.0.0.0/24",
    "10.5.0.0/24"
  ]
}

