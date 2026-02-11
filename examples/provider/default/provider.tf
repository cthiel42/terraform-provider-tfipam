// By default, the provider will store data at 
// .terraform/ipam-storage.json in the current working
// directory. The file_storage example show how to
// customize this location. 

terraform {
  required_providers {
    tfipam = {
      source  = "cthiel42/tfipam"
      version = "1.2.0"
    }
  }
}

provider "tfipam" {}
