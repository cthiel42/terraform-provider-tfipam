package storage

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("not found")
)

type Pool struct {
	Name  string   `json:"name"`
	CIDRs []string `json:"cidrs"`
}

type Allocation struct {
	ID            string `json:"id"`
	PoolName      string `json:"pool_name"`
	AllocatedCIDR string `json:"allocated_cidr"`
	PrefixLength  int    `json:"prefix_length"`
}

type Storage interface {
	// pool operations
	GetPool(ctx context.Context, name string) (*Pool, error)
	ListPools(ctx context.Context) ([]Pool, error)
	SavePool(ctx context.Context, pool *Pool) error
	DeletePool(ctx context.Context, name string) error

	// allocation operations
	GetAllocation(ctx context.Context, id string) (*Allocation, error)
	ListAllocations(ctx context.Context) ([]Allocation, error)
	ListAllocationsByPool(ctx context.Context, poolName string) ([]Allocation, error)
	SaveAllocation(ctx context.Context, allocation *Allocation) error
	DeleteAllocation(ctx context.Context, id string) error

	Close() error
}

type Config struct {
	Type string // "file", "azure_blob", "aws_s3"

	// File backend config
	FilePath string

	// Azure Blob Storage config
	AzureConnectionString string
	AzureContainerName    string
	AzureBlobName         string

	// AWS S3 Storage config
	S3Region          string
	S3BucketName      string
	S3ObjectKey       string
	S3AccessKeyID     string // Optional: uses default credential chain if empty
	S3SecretAccessKey string // Optional: required if S3AccessKeyID is provided
	S3SessionToken    string // Optional: for temporary credentials
}

func Factory(ctx context.Context, config *Config) (Storage, error) {
	switch config.Type {
	case "file", "": // default to file
		return NewFileStorage(config.FilePath)
	case "azure_blob":
		return NewAzureBlobStorage(config.AzureConnectionString, config.AzureContainerName, config.AzureBlobName)
	case "aws_s3":
		return NewS3Storage(config.S3Region, config.S3BucketName, config.S3ObjectKey,
			config.S3AccessKeyID, config.S3SecretAccessKey, config.S3SessionToken)
	default:
		return nil, errors.New("unknown storage type")
	}
}
