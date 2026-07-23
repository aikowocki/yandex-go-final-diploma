package objectstore

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/contracts"
)

// bucketCheckMaxAttempts/bucketCheckBaseDelay — ретрай проверки/создания бакета при старте.
const (
	bucketCheckMaxAttempts = 5
	bucketCheckBaseDelay   = 500 * time.Millisecond
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

	if err := ensureBucket(ctx, client, cfg.Bucket); err != nil {
		return nil, err
	}

	return &Store{client: client, bucket: cfg.Bucket}, nil
}

// ensureBucket проверяет наличие бакета и создаёт его при отсутствии, повторяя обе операции
// при ошибке (см. bucketCheckMaxAttempts) — сразу после старта MinIO может недолго отвечать
// "Server not initialized yet" даже если health-check контейнера уже зелёный.
func ensureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	var lastErr error
	delay := bucketCheckBaseDelay
	for attempt := 0; attempt < bucketCheckMaxAttempts; attempt++ {
		lastErr = checkAndCreateBucket(ctx, client, bucket)
		if lastErr == nil {
			return nil
		}
		if attempt == bucketCheckMaxAttempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay *= 2
		}
	}
	return lastErr
}

// checkAndCreateBucket — одна попытка BucketExists + (при отсутствии) MakeBucket.
func checkAndCreateBucket(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("objectstore: check bucket: %w", err)
	}
	if exists {
		return nil
	}
	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("objectstore: create bucket: %w", err)
	}
	return nil
}
