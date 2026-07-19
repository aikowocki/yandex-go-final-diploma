package grpcclient

import (
	"context"
	"errors"
	"fmt"
	"io"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
)

const uploadChunkSize = 64 * 1024

// UploadBlob стримит содержимое r на сервер чанками и возвращает blob-ref/размер.
func (c *Client) UploadBlob(ctx context.Context, accessToken, secretID string, r io.Reader) (string, int64, error) {
	ctx = withBearer(ctx, accessToken)
	stream, err := c.blob.UploadBlob(ctx)
	if err != nil {
		return "", 0, mapErr(err)
	}

	buf := make([]byte, uploadChunkSize)
	first := true
	for {
		n, rerr := r.Read(buf)
		if n > 0 {
			chunk := &pb.UploadBlobChunk{Data: append([]byte(nil), buf[:n]...)}
			if first {
				chunk.SecretId = secretID
				first = false
			}
			if serr := stream.Send(chunk); serr != nil {
				return "", 0, mapErr(serr)
			}
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			return "", 0, fmt.Errorf("grpcclient: read blob data: %w", rerr)
		}
	}

	// Поток может быть пустым (нулевой файл) — сервер всё равно должен получить secret_id хотя
	// бы одним сообщением.
	if first {
		if serr := stream.Send(&pb.UploadBlobChunk{SecretId: secretID}); serr != nil {
			return "", 0, mapErr(serr)
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		return "", 0, mapErr(err)
	}
	return res.GetBlobRef(), res.GetBlobSize(), nil
}

// DownloadBlob открывает стрим скачивания бинарного секрета с сервера.
func (c *Client) DownloadBlob(ctx context.Context, accessToken, secretID string) (io.ReadCloser, error) {
	ctx = withBearer(ctx, accessToken)
	stream, err := c.blob.DownloadBlob(ctx, &pb.DownloadBlobRequest{SecretId: secretID})
	if err != nil {
		return nil, mapErr(err)
	}
	return &blobStreamReader{stream: stream}, nil
}

// AttachBlob привязывает загруженный blob к секрету с оптимистичной блокировкой по версии.
func (c *Client) AttachBlob(ctx context.Context, accessToken, secretID string, baseVersion int64, blobRef string, blobSize int64) (int64, error) {
	ctx = withBearer(ctx, accessToken)
	resp, err := c.blob.AttachBlob(ctx, &pb.AttachBlobRequest{
		SecretId:    secretID,
		BaseVersion: baseVersion,
		BlobRef:     blobRef,
		BlobSize:    blobSize,
	})
	if err != nil {
		if conflict := conflictFromStatus(err); conflict != nil {
			return 0, conflict
		}
		return 0, mapErr(err)
	}
	return resp.GetVersion(), nil
}

type blobStreamReader struct {
	stream pb.BlobService_DownloadBlobClient
	buf    []byte
}

func (r *blobStreamReader) Read(p []byte) (int, error) {
	for len(r.buf) == 0 {
		chunk, err := r.stream.Recv()
		if errors.Is(err, io.EOF) {
			return 0, io.EOF
		}
		if err != nil {
			return 0, mapErr(err)
		}
		r.buf = chunk.GetData()
	}
	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}

func (r *blobStreamReader) Close() error { return nil }
