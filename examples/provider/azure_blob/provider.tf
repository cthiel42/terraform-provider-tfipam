terraform {
  required_providers {
    tfipam = {
      source  = "cthiel42/tfipam"
      version = "1.1.0"
    }
  }
}

provider "tfipam" {
  storage_type            = "azure_blob"
  azure_connection_string = "DefaultEndpointsProtocol=https;AccountName=myaccount;AccountKey=mykey;EndpointSuffix=core.windows.net"
  azure_container_name    = "tfipam"
  azure_blob_name         = "ipam-storage.json" # Optional: defaults to "ipam-storage.json"
}

resource "tfipam_pool" "example" {
  name = "pool_example"
  cidrs = [
    "10.0.0.0/24",
    "10.5.0.0/24"
  ]
}
