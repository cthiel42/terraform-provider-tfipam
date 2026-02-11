# Terraform IP Address Manager (tfipam)

A lightweight Terraform provider for managing IP address pools and allocations. The TFIPAM Provider lets you declare and manage IP pools, allocate and release addresses through Terraform, and persist configurations to a storage backend (default: file). It is intended for simple IPAM workflows and automation in local or small-scale environments.

Features
- Create, update, and delete IP pools and prefixes
- Allocate and release individual IP addresses with predictable, idempotent behavior
- Configurable storage backend (file-based storage by default)
- Simple schema and minimal external dependencies — ideal for local development and CI workflows

Pool resources specify a CIDR to disperse IP's from, and allocation resources are used to dynamically allocate IP's or subnets from the pool.

Quick example
```hcl
terraform {
  required_providers {
    tfipam = {
      source = "cthiel42/tfipam"
      version = "1.1.0"
    }
  }
}

provider "tfipam" {
  storage_type = "file"
  file_path    = ".terraform/ipam-storage.json"
}

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
```

Allocation resources provision CIDRs from the pool based on a greedy search and are stored in the `allocated_cidr` field. Data calls can also be used to read this information about allocations.

Data Call Example
```hcl
data "tfipam_allocation" "example" {
  id        = "allocation_example_0"
  pool_name = "pool_example"
}
```

## Provider Configuration

This provider stores pool and allocation information in a separate file from Terraform's state. This is due to limitations within Terraform when accessing information about other resource's state, which is a core requirement for a parent-child resource relationship similar to what is implemented in this provider. There's currently a few storage backends implemented for this purpose. Their example configurations are detailed below.


### File (Default)
The file backend is the default backend. If you do not pass any parameters to the provider, it will store information in a file at `.terraform/ipam-storage.json` from the current working directory. To customize the location of the file, you can use a configuration similar to below.
```hcl
provider "tfipam" {
  storage_type = "file"
  file_path    = "ipam_storage_example.json"
}
```

### AWS S3
This will store a json file in the configured AWS S3 bucket. You can either explicity specify credentials for the provider to use, or rely on the SDK to determine them through ~/.aws/credentials or environment variables.

**Credentials Declared Explicitly**
```hcl
provider "tfipam" {
  storage_type         = "aws_s3"
  s3_region            = "us-east-1"
  s3_bucket_name       = "my-tfipam-bucket"
  s3_object_key        = "ipam-storage.json" # Optional: defaults to "ipam-storage.json"
  s3_access_key_id     = "AKIAABCDEFGHEXAMPLE"
  s3_secret_access_key = "ACCESSKEYEXAMPLE1234567890"
  s3_endpoint_url      = "https://s3.example.com" # Optional: for S3 compatible services like MinIO or LocalStack
  # s3_session_token    = "token"                 # Optional: for temporary credentials
}
```

**Using Default AWS Credential Chain (env vars, ~/.aws/credentials, etc)**
```hcl
provider "tfipam" {
  storage_type   = "aws_s3"
  s3_region      = "us-east-1"
  s3_bucket_name = "my-tfipam-bucket"
  s3_object_key  = "ipam-storage.json"
}
```

### Azure
This will store a json file in the configured Azure Blob Container.
```hcl
provider "tfipam" {
  storage_type            = "azure_blob"
  azure_connection_string = "DefaultEndpointsProtocol=https;AccountName=myaccount;AccountKey=mykey;EndpointSuffix=core.windows.net"
  azure_container_name    = "tfipam"
  azure_blob_name         = "ipam-storage.json" # Optional: defaults to "ipam-storage.json"
}
```

## Folder Structure

- `examples/` contains helpful examples to get you started
- `internal/` contains the source source for the provider
- `docs/` contains the markdown files used on the Terraform registry

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install .
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.


## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install .`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

```shell
make testacc
```
