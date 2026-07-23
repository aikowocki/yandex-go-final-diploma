package grpcserver

import (
	"context"
	"errors"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver/interceptor"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver/mapper"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/blob"
)

// UploadBlob — client-streaming RPC: принимает чанки бинарного секрета от клиента и сохраняет их в хранилище.
func (s *Server) UploadBlob(stream pb.BlobService_UploadBlobServer) error {
	if s.blob == nil {
		return mapBlobErr(blob.ErrBlobStorageDisabled)
	}

	ctx := stream.Context()
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return errNoUser()
	}

	first, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return status.Error(codes.InvalidArgument, "empty upload stream")
		}
		return status.Error(codes.Internal, "internal error")
	}
	secretID := first.GetSecretId()

	pr, pw := io.Pipe()
	recvErrCh := make(chan error, 1)
	go func() {
		defer func() { _ = pw.Close() }()
		if _, werr := pw.Write(first.GetData()); werr != nil {
			recvErrCh <- werr
			return
		}
		for {
			chunk, rerr := stream.Recv()
			if errors.Is(rerr, io.EOF) {
				recvErrCh <- nil
				return
			}
			if rerr != nil {
				recvErrCh <- rerr
				return
			}
			if _, werr := pw.Write(chunk.GetData()); werr != nil {
				recvErrCh <- werr
				return
			}
		}
	}()

	blobRef, size, err := s.blob.UploadChunked(ctx, userID, secretID, pr)
	if err != nil {
		return mapBlobErr(err)
	}
	if recvErr := <-recvErrCh; recvErr != nil && !errors.Is(recvErr, io.ErrClosedPipe) {
		return status.Error(codes.Internal, "internal error")
	}

	return stream.SendAndClose(&pb.UploadBlobResult{BlobRef: blobRef, BlobSize: size})
}

// DownloadBlob — server-streaming RPC: читает объект из хранилища и отдаёт его клиенту чанками.
func (s *Server) DownloadBlob(req *pb.DownloadBlobRequest, stream pb.BlobService_DownloadBlobServer) error {
	if s.blob == nil {
		return mapBlobErr(blob.ErrBlobStorageDisabled)
	}

	ctx := stream.Context()
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return errNoUser()
	}

	rc, err := s.blob.DownloadChunked(ctx, userID, req.GetSecretId())
	if err != nil {
		return mapBlobErr(err)
	}
	defer func() { _ = rc.Close() }()

	buf := make([]byte, downloadChunkSize)
	for {
		n, rerr := rc.Read(buf)
		if n > 0 {
			if serr := stream.Send(&pb.DownloadBlobChunk{Data: append([]byte(nil), buf[:n]...)}); serr != nil {
				return status.Error(codes.Internal, "internal error")
			}
		}
		if errors.Is(rerr, io.EOF) {
			return nil
		}
		if rerr != nil {
			return status.Error(codes.Internal, "internal error")
		}
	}
}

// downloadChunkSize — размер чанка при отдаче блоба клиенту (не связан с размером чанков
// потокового AEAD клиента — тот определяется клиентом при шифровании).
const downloadChunkSize = 64 * 1024

// AttachBlob привязывает загруженный blob (по ссылке и размеру) к секрету и поднимает версию.
func (s *Server) AttachBlob(ctx context.Context, req *pb.AttachBlobRequest) (*pb.AttachBlobResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, errNoUser()
	}

	version, err := s.secret.AttachBlob(ctx, mapper.AttachBlobParams(userID, req))
	if err != nil {
		return nil, mapSecretErr(err)
	}
	return &pb.AttachBlobResponse{Version: version}, nil
}

// mapBlobErr преобразует ошибки usecase/blob в gRPC status-коды.
func mapBlobErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, blob.ErrBlobStorageDisabled):
		return status.Error(codes.Unimplemented, "binary secrets are not supported: object storage is not configured")
	case errors.Is(err, blob.ErrSecretNotFound):
		return status.Error(codes.NotFound, "secret not found")
	case errors.Is(err, blob.ErrEmptySecretID), errors.Is(err, blob.ErrNoData):
		return status.Error(codes.InvalidArgument, "invalid blob request")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
