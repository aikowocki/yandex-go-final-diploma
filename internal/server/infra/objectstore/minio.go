package objectstore

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/contracts"
)

// Store реализует contracts.BlobStorage поверх MinIO/S3.
type Store struct {
	client *minio.Client
	bucket string
}

var _ contracts.BlobStorage = (*Store)(nil)

// Config описывает параметры подключения к объектному хранилищу.
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// New создаёт клиент MinIO и при необходимости создаёт бакет.
func New(ctx context.Context, cfg Config) (*Store, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("objectstore: create client: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("objectstore: check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("objectstore: create bucket: %w", err)
		}
	}

	return &Store{client: client, bucket: cfg.Bucket}, nil
}
