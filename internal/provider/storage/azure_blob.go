package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
)

type AzureBlobStorage struct {
	client        *azblob.Client
	containerName string
	blobName      string
	mu            sync.RWMutex
	data          *blobData
}

type blobData struct {
	Pools       map[string]*Pool       `json:"pools"`
	Allocations map[string]*Allocation `json:"allocations"`
}

// NewAzureBlobStorage creates a new Azure Blob Storage backend
// connectionString: Azure Storage connection string
// containerName: Name of the blob container
// blobName: Name of the blob file (e.g., "ipam-storage.json")
func NewAzureBlobStorage(connectionString, containerName, blobName string) (*AzureBlobStorage, error) {
	if connectionString == "" {
		return nil, errors.New("azure connection string is required")
	}
	if containerName == "" {
		return nil, errors.New("azure container name is required")
	}
	if blobName == "" {
		blobName = "ipam-storage.json"
	}

	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure blob client: %w", err)
	}

	abs := &AzureBlobStorage{
		client:        client,
		containerName: containerName,
		blobName:      blobName,
		data: &blobData{
			Pools:       make(map[string]*Pool),
			Allocations: make(map[string]*Allocation),
		},
	}

	// try to load existing data
	ctx := context.Background()
	if err := abs.load(ctx); err != nil {
		// If blob doesn't exist, that's okay - we'll create it on first save
		if !bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil, fmt.Errorf("failed to load storage blob: %w", err)
		}
	}

	return abs, nil
}

func (abs *AzureBlobStorage) load(ctx context.Context) error {
	abs.mu.Lock()
	defer abs.mu.Unlock()

	downloadResponse, err := abs.client.DownloadStream(ctx, abs.containerName, abs.blobName, nil)
	if err != nil {
		return err
	}
	defer downloadResponse.Body.Close()

	data, err := io.ReadAll(downloadResponse.Body)
	if err != nil {
		return fmt.Errorf("failed to read blob data: %w", err)
	}

	return json.Unmarshal(data, abs.data)
}

func (abs *AzureBlobStorage) save(ctx context.Context) error {
	data, err := json.MarshalIndent(abs.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal storage data: %w", err)
	}

	_, err = abs.client.UploadStream(ctx, abs.containerName, abs.blobName,
		bytes.NewReader(data), nil)
	if err != nil {
		return fmt.Errorf("failed to upload blob: %w", err)
	}

	return nil
}

func (abs *AzureBlobStorage) GetPool(ctx context.Context, name string) (*Pool, error) {
	abs.mu.RLock()
	defer abs.mu.RUnlock()

	pool, exists := abs.data.Pools[name]
	if !exists {
		return nil, ErrNotFound
	}

	// return copy
	poolCopy := *pool
	return &poolCopy, nil
}

func (abs *AzureBlobStorage) ListPools(ctx context.Context) ([]Pool, error) {
	abs.mu.RLock()
	defer abs.mu.RUnlock()

	// return copies
	pools := make([]Pool, 0, len(abs.data.Pools))
	for _, pool := range abs.data.Pools {
		pools = append(pools, *pool)
	}

	return pools, nil
}

func (abs *AzureBlobStorage) SavePool(ctx context.Context, pool *Pool) error {
	abs.mu.Lock()
	defer abs.mu.Unlock()

	// save a copy
	poolCopy := *pool
	abs.data.Pools[pool.Name] = &poolCopy

	return abs.save(ctx)
}

func (abs *AzureBlobStorage) DeletePool(ctx context.Context, name string) error {
	abs.mu.Lock()
	defer abs.mu.Unlock()

	if _, exists := abs.data.Pools[name]; !exists {
		return ErrNotFound
	}

	delete(abs.data.Pools, name)
	return abs.save(ctx)
}

func (abs *AzureBlobStorage) GetAllocation(ctx context.Context, id string) (*Allocation, error) {
	abs.mu.RLock()
	defer abs.mu.RUnlock()

	allocation, exists := abs.data.Allocations[id]
	if !exists {
		return nil, ErrNotFound
	}

	// return copy
	allocCopy := *allocation
	return &allocCopy, nil
}

func (abs *AzureBlobStorage) ListAllocations(ctx context.Context) ([]Allocation, error) {
	abs.mu.RLock()
	defer abs.mu.RUnlock()

	// return copies
	allocations := make([]Allocation, 0, len(abs.data.Allocations))
	for _, alloc := range abs.data.Allocations {
		allocations = append(allocations, *alloc)
	}

	return allocations, nil
}

func (abs *AzureBlobStorage) ListAllocationsByPool(ctx context.Context, poolName string) ([]Allocation, error) {
	abs.mu.RLock()
	defer abs.mu.RUnlock()

	allocations := make([]Allocation, 0)
	for _, alloc := range abs.data.Allocations {
		if alloc.PoolName == poolName {
			allocations = append(allocations, *alloc)
		}
	}

	return allocations, nil
}

func (abs *AzureBlobStorage) SaveAllocation(ctx context.Context, allocation *Allocation) error {
	abs.mu.Lock()
	defer abs.mu.Unlock()

	allocCopy := *allocation
	abs.data.Allocations[allocation.ID] = &allocCopy

	return abs.save(ctx)
}

func (abs *AzureBlobStorage) DeleteAllocation(ctx context.Context, id string) error {
	abs.mu.Lock()
	defer abs.mu.Unlock()

	if _, exists := abs.data.Allocations[id]; !exists {
		return ErrNotFound
	}

	delete(abs.data.Allocations, id)
	return abs.save(ctx)
}

func (abs *AzureBlobStorage) Close() error {
	// Azure SDK doesn't require explicit cleanup
	return nil
}