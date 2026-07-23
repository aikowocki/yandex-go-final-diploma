package cli

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

// BuildInfo — версия и дата сборки, прокидываются из main через kong-bind.
type BuildInfo struct {
	Version string
	Date    string
}

// PingCmd — проверка связности с сервером.
type PingCmd struct{}

// Run проверяет связность с сервером через gRPC Ping.
func (c *PingCmd) Run(client *grpcclient.Client) error {
	msg, err := client.Ping(context.Background())
	if err != nil {
		return err
	}
	fmt.Println(msg)
	return nil
}

// VersionCmd — печать версии клиента.
type VersionCmd struct{}

// Run печатает версию и дату сборки клиента.
func (c *VersionCmd) Run(info *BuildInfo) error {
	fmt.Printf("gophkeeper-client %s (built %s)\n", info.Version, info.Date)
	return nil
}
