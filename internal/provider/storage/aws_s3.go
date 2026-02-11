package storage

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Storage struct {
	client     *s3.Client
	bucketName string
	objectKey  string
	mu         sync.RWMutex
	data       *s3Data
}

type s3Data struct {
	Pools       map[string]*Pool       `json:"pools"`
	Allocations map[string]*Allocation `json:"allocations"`
}

// NewS3Storage creates a new AWS S3 Storage backend
// region: AWS region (e.g. "us-east-1")
// bucketName: Name of the S3 bucket
// objectKey: S3 object key (path to the JSON file, e.g. "ipam-storage.json")
// accessKeyID: AWS Access Key ID (optional, uses default credential chain if empty)
// secretAccessKey: AWS Secret Access Key (optional, required if accessKeyID is provided)
// sessionToken: AWS Session Token (optional, for temporary credentials)
// endpointURL: Custom S3 endpoint URL (optional, for S3 compatible services like MinIO or LocalStack)
// skipTLSVerify: Skip TLS certificate verification (optional)
func NewS3Storage(region, bucketName, objectKey, accessKeyID, secretAccessKey, sessionToken, endpointURL string, skipTLSVerify bool) (*S3Storage, error) {
	if region == "" {
		return nil, errors.New("aws region is required")
	}
	if bucketName == "" {
		return nil, errors.New("s3 bucket name is required")
	}
	if objectKey == "" {
		objectKey = "ipam-storage.json"
	}

	if accessKeyID != "" && secretAccessKey == "" {
		return nil, errors.New("aws secret access key is required when access key id is provided")
	}
	if accessKeyID == "" && secretAccessKey != "" {
		return nil, errors.New("aws access key id is required when secret access key is provided")
	}

	ctx := context.Background()
	var cfg aws.Config
	var err error

	// load config with credentials if provided otherwise use default config
	if accessKeyID != "" && secretAccessKey != "" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     accessKeyID,
					SecretAccessKey: secretAccessKey,
					SessionToken:    sessionToken,
				}, nil
			})),
		)
	} else {
		// Use default credential chain (env vars, ~/.aws/credentials, IAM role, etc)
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(region))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	// create s3 client with custom endpoint if provided
	var client *s3.Client
	if endpointURL != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpointURL)
			o.UsePathStyle = true // uses path style addressing where the bucket name is part of the url path, not subdomain. required for most s3 compatible services

			// Skip TLS verification
			if skipTLSVerify {
				o.HTTPClient = &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
					},
				}
			}
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}

	s3s := &S3Storage{
		client:     client,
		bucketName: bucketName,
		objectKey:  objectKey,
		data: &s3Data{
			Pools:       make(map[string]*Pool),
			Allocations: make(map[string]*Allocation),
		},
	}

	// try to load existing data. If object doesn't exist, it'll be created on first save
	if err := s3s.load(ctx); err != nil {
		var nsk *types.NoSuchKey
		if !errors.As(err, &nsk) {
			return nil, fmt.Errorf("failed to load storage object: %w", err)
		}
	}

	return s3s, nil
}

func (s3s *S3Storage) load(ctx context.Context) error {
	s3s.mu.Lock()
	defer s3s.mu.Unlock()

	result, err := s3s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3s.bucketName),
		Key:    aws.String(s3s.objectKey),
	})
	if err != nil {
		return err
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return fmt.Errorf("failed to read s3 object data: %w", err)
	}

	return json.Unmarshal(data, s3s.data)
}

func (s3s *S3Storage) save(ctx context.Context) error {
	data, err := json.MarshalIndent(s3s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal storage data: %w", err)
	}

	_, err = s3s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s3s.bucketName),
		Key:    aws.String(s3s.objectKey),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload s3 object: %w", err)
	}

	return nil
}

func (s3s *S3Storage) GetPool(ctx context.Context, name string) (*Pool, error) {
	s3s.mu.RLock()
	defer s3s.mu.RUnlock()

	pool, exists := s3s.data.Pools[name]
	if !exists {
		return nil, ErrNotFound
	}

	// return copy
	poolCopy := *pool
	return &poolCopy, nil
}

func (s3s *S3Storage) ListPools(ctx context.Context) ([]Pool, error) {
	s3s.mu.RLock()
	defer s3s.mu.RUnlock()

	// return copies
	pools := make([]Pool, 0, len(s3s.data.Pools))
	for _, pool := range s3s.data.Pools {
		pools = append(pools, *pool)
	}

	return pools, nil
}

func (s3s *S3Storage) SavePool(ctx context.Context, pool *Pool) error {
	s3s.mu.Lock()
	defer s3s.mu.Unlock()

	// save a copy
	poolCopy := *pool
	s3s.data.Pools[pool.Name] = &poolCopy

	return s3s.save(ctx)
}

func (s3s *S3Storage) DeletePool(ctx context.Context, name string) error {
	s3s.mu.Lock()
	defer s3s.mu.Unlock()

	if _, exists := s3s.data.Pools[name]; !exists {
		return ErrNotFound
	}

	delete(s3s.data.Pools, name)
	return s3s.save(ctx)
}

func (s3s *S3Storage) GetAllocation(ctx context.Context, id string) (*Allocation, error) {
	s3s.mu.RLock()
	defer s3s.mu.RUnlock()

	allocation, exists := s3s.data.Allocations[id]
	if !exists {
		return nil, ErrNotFound
	}

	// return copy
	allocCopy := *allocation
	return &allocCopy, nil
}

func (s3s *S3Storage) ListAllocations(ctx context.Context) ([]Allocation, error) {
	s3s.mu.RLock()
	defer s3s.mu.RUnlock()

	// return copies
	allocations := make([]Allocation, 0, len(s3s.data.Allocations))
	for _, alloc := range s3s.data.Allocations {
		allocations = append(allocations, *alloc)
	}

	return allocations, nil
}

func (s3s *S3Storage) ListAllocationsByPool(ctx context.Context, poolName string) ([]Allocation, error) {
	s3s.mu.RLock()
	defer s3s.mu.RUnlock()

	allocations := make([]Allocation, 0)
	for _, alloc := range s3s.data.Allocations {
		if alloc.PoolName == poolName {
			allocations = append(allocations, *alloc)
		}
	}

	return allocations, nil
}

func (s3s *S3Storage) SaveAllocation(ctx context.Context, allocation *Allocation) error {
	s3s.mu.Lock()
	defer s3s.mu.Unlock()

	// save a copy
	allocCopy := *allocation
	s3s.data.Allocations[allocation.ID] = &allocCopy

	return s3s.save(ctx)
}

func (s3s *S3Storage) DeleteAllocation(ctx context.Context, id string) error {
	s3s.mu.Lock()
	defer s3s.mu.Unlock()

	if _, exists := s3s.data.Allocations[id]; !exists {
		return ErrNotFound
	}

	delete(s3s.data.Allocations, id)
	return s3s.save(ctx)
}

func (s3s *S3Storage) Close() error {
	// AWS SDK doesn't require explicit cleanup
	return nil
}
