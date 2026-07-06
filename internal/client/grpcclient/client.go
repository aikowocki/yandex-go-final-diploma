package grpcclient

import (
	"context"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client — gRPC-клиент
type Client struct {
	conn *grpc.ClientConn
	auth pb.AuthServiceClient
}

// New создаёт подключение к серверу.
func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn: conn,
		auth: pb.NewAuthServiceClient(conn),
	}, nil
}

// Close закрывает соединение.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Ping проверяет связность с сервером.
func (c *Client) Ping(ctx context.Context) (string, error) {
	resp, err := c.auth.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		return "", err
	}
	return resp.Message, nil
}
