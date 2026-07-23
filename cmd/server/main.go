package main

import (
	"context"
	"log"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/app"
)

func main() {
	ctx := context.Background()

	container, err := app.New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := container.Run(); err != nil {
		log.Fatal(err)
	}
}
