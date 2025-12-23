terraform {
  required_providers {
    ipam = {
      source = "hashicorp.com/edu/tf-ipam"
    }
  }
}

provider "ipam" {
  storage_type = "file"
  file_path    = "ipam_storage_example.json"
}

resource "ipam_pool" "example" {
  name = "pool_example"
  cidrs = [
    "10.0.0.0/24",
    "10.5.0.0/24"
  ]
}

