package providers

import "github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver"

func NewGRPC() *grpcserver.Server {
	return grpcserver.New()
}
