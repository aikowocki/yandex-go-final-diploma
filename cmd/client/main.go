package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
)

var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("gophkeeper-client %s (built %s)\n", version, formatBuildDate(buildDate))
		return
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func formatBuildDate(raw string) string {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return t.Format("2006-01-02")
}
