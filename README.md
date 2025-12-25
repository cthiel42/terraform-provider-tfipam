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
      version = "1.0.3"
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

This provider requires a backend for IPAM information to be stored. While I sought out to store everything in Terraform's state, I found limitations within Terraform that prevented this from being a reality with parent-child resources like this provider implements. As a result, having a backend to store this information was a necessity. I have made an attempt to have a few commonly used backends built in. If there's other backends that would be useful, please open a GitHub issue with your suggestions. I also welcome PR's.


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
