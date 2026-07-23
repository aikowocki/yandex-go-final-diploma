// cmd/autoupdater — создаёт секрет AUTOUPDATED и обновляет его каждые 5 секунд.
// Используется для тестирования real-time sync между двумя клиентами.
//
// Предусловия:
//   - Сервер запущен, пользователь test/test зарегистрирован, шифрование настроено
//   - Vault "Personal" уже существует (из seed)
//
// Запуск:
//
//	go run ./cmd/autoupdater
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

const (
	secretTitle = "AUTOUPDATED"
	vaultName   = "Personal"
	updateEvery = 5 * time.Second
)

func main() {
	dataDir, err := config.DefaultDataDir()
	if err != nil {
		log.Fatal(err)
	}

	cfg := &config.ClientConfig{
		ServerAddr:      "localhost:9090",
		DataDir:         dataDir,
		LogLevel:        "info",
		Lang:            "en",
		AutolockMinutes: 0,
		TOTPRevealMode:  "focused",
	}

	container, err := app.New(cfg)
	if err != nil {
		log.Fatalf("app.New: %v", err)
	}
	defer func() { _ = container.Close() }()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Login + Unlock
	fmt.Println("Logging in as test/test...")
	if err := container.Auth.Login(ctx, "test", []byte("test")); err != nil {
		log.Fatalf("Login: %v", err)
	}
	if err := container.Auth.Unlock(ctx, []byte("test")); err != nil {
		log.Fatalf("Unlock: %v", err)
	}

	// Find vault "Personal"
	vaults, err := container.Vault.List(ctx)
	if err != nil {
		log.Fatalf("List vaults: %v", err)
	}
	var vaultID string
	for _, v := range vaults {
		if v.Name == vaultName {
			vaultID = v.ID
			break
		}
	}
	if vaultID == "" {
		log.Fatalf("Vault %q not found. Run seed first.", vaultName)
	}
	fmt.Printf("Using vault %s (%s)\n", vaultName, vaultID)

	// Create initial secret
	counter := 0
	password := fmt.Sprintf("value-%d-%s", counter, time.Now().Format("15:04:05"))
	secretID, err := container.Secret.CreateLoginPassword(ctx, vaultID, secret.CreateLoginPasswordInput{
		Title:    secretTitle,
		Username: "auto-updater",
		Password: password,
		URI:      "https://autoupdater.test",
		Tags:     []string{"test", "auto"},
		Note:     fmt.Sprintf("Created at %s", time.Now().Format(time.RFC3339)),
	})
	if err != nil {
		log.Fatalf("Create secret: %v", err)
	}
	fmt.Printf("Created secret %s = %q\n", secretID, password)

	// Get initial version
	ver, ok, err := container.Secret.LocalVersion(ctx, secretID)
	if err != nil || !ok {
		log.Fatalf("Get version: %v (ok=%v)", err, ok)
	}

	// Update loop
	ticker := time.NewTicker(updateEvery)
	defer ticker.Stop()

	fmt.Printf("Updating every %s. Ctrl+C to stop.\n\n", updateEvery)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nStopped.")
			return
		case <-ticker.C:
			counter++
			password = fmt.Sprintf("value-%d-%s", counter, time.Now().Format("15:04:05"))

			conflict, err := container.Secret.UpdateLoginPassword(ctx, vaultID, secretID, ver, secret.CreateLoginPasswordInput{
				Title:    secretTitle,
				Username: "auto-updater",
				Password: password,
				URI:      "https://autoupdater.test",
				Tags:     []string{"test", "auto"},
				Note:     fmt.Sprintf("Update #%d at %s", counter, time.Now().Format(time.RFC3339)),
			})
			if err != nil {
				fmt.Printf("  ERROR update #%d: %v\n", counter, err)
				continue
			}
			if conflict != nil {
				fmt.Printf("  CONFLICT at update #%d (ver %d)\n", counter, ver)
				// Принимаем серверную версию и продолжаем.
				ver = conflict.ServerVersion
				continue
			}
			ver++
			fmt.Printf("  [%s] update #%d → %q (ver=%d)\n", time.Now().Format("15:04:05"), counter, password, ver)
		}
	}
}
