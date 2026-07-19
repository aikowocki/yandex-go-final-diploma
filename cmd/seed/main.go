// cmd/seed — генератор тестовых данных для GophKeeper (массовый).
// Создаёт 5 vault'ов и заполняет каждый 50+ секретами всех типов.
//
// Предусловия:
//   - Сервер запущен (docker compose up)
//   - Пользователь test/test уже зарегистрирован и шифрование настроено
//   - Мастер-пароль: test
//
// Запуск:
//
//	go run ./cmd/seed
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

var (
	adjectives = []string{"Fast", "Secure", "Dark", "Light", "Strong", "Hidden", "Golden", "Silver", "Crystal", "Shadow"}
	nouns      = []string{"Pass", "Key", "Lock", "Gate", "Wall", "Shield", "Token", "Code", "Vault", "Safe"}
	domains    = []string{"github.com", "gitlab.com", "google.com", "aws.amazon.com", "digitalocean.com", "heroku.com", "vercel.app", "netlify.com", "cloudflare.com", "docker.com"}
	banks      = []string{"Tinkoff", "Sberbank", "Alfa-Bank", "VTB", "Raiffeisen", "Gazprom", "Ozon Bank", "Yandex Pay"}
	brands     = []string{"Visa", "Mastercard", "Mir", "AmEx", "UnionPay"}
	issuers    = []string{"GitHub", "GitLab", "AWS", "Google", "Microsoft", "Dropbox", "Slack", "Discord", "Twitter", "Facebook"}
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
		Lang:            "ru",
		AutolockMinutes: 5,
		TOTPRevealMode:  "focused",
	}

	container, err := app.New(cfg)
	if err != nil {
		log.Fatalf("app.New: %v", err)
	}
	defer func() { _ = container.Close() }()

	ctx := context.Background()

	// --- Login ---
	fmt.Println("Logging in as test/test...")
	if err := container.Auth.Login(ctx, "test", []byte("test")); err != nil {
		log.Fatalf("Login: %v", err)
	}

	// --- Unlock (master key = "test") ---
	fmt.Println("Unlocking with master key 'test'...")
	if err := container.Auth.Unlock(ctx, []byte("test")); err != nil {
		log.Fatalf("Unlock: %v", err)
	}

	// --- Create Vaults ---
	vaultNames := []string{"Personal", "Work", "Finance", "Development", "Social"}
	vaultIDs := make([]string, 0, len(vaultNames))

	for _, name := range vaultNames {
		fmt.Printf("Creating vault: %s\n", name)
		id, err := container.Vault.Create(ctx, name)
		if err != nil {
			log.Fatalf("Create vault %s: %v", name, err)
		}
		vaultIDs = append(vaultIDs, id)
	}

	totalCreated := 0

	// --- Mass-create secrets per vault ---
	for vi, vaultID := range vaultIDs {
		vaultName := vaultNames[vi]
		fmt.Printf("\n=== Vault: %s ===\n", vaultName)

		// 30 logins per vault
		for i := 0; i < 30; i++ {
			title := fmt.Sprintf("%s %s %d", pick(adjectives), pick(nouns), i+1)
			domain := pick(domains)
			_, err := container.Secret.CreateLoginPassword(ctx, vaultID, secret.CreateLoginPasswordInput{
				Title:    title,
				URI:      fmt.Sprintf("https://%s/app/%d", domain, i),
				Username: fmt.Sprintf("user_%s_%d@%s", strings.ToLower(vaultName), i, domain),
				Password: fmt.Sprintf("%s-%s-%04d!", pick(adjectives), pick(nouns), rand.Intn(10000)),
				Tags:     randomTags(),
				Note:     fmt.Sprintf("Auto-generated login #%d for %s", i+1, vaultName),
			})
			if err != nil {
				log.Printf("  WARN login %d: %v", i, err)
			} else {
				totalCreated++
			}
		}
		fmt.Printf("  + 30 logins\n")

		// 10 notes per vault
		for i := 0; i < 10; i++ {
			body := strings.Repeat(fmt.Sprintf("Line %d of note %d in %s.\n", i, i+1, vaultName), 5)
			_, err := container.Secret.CreateText(ctx, vaultID, secret.CreateTextInput{
				Title: fmt.Sprintf("Note: %s %s %d", pick(adjectives), pick(nouns), i+1),
				Body:  body,
				Tags:  randomTags(),
				Note:  fmt.Sprintf("Test note #%d", i+1),
			})
			if err != nil {
				log.Printf("  WARN note %d: %v", i, err)
			} else {
				totalCreated++
			}
		}
		fmt.Printf("  + 10 notes\n")

		// 5 cards per vault
		for i := 0; i < 5; i++ {
			pan := fmt.Sprintf("4276%012d", rand.Int63n(1000000000000))
			_, err := container.Secret.CreateBankCard(ctx, vaultID, secret.CreateBankCardInput{
				Title:      fmt.Sprintf("%s Card %d", pick(banks), i+1),
				Bank:       pick(banks),
				Cardholder: fmt.Sprintf("TEST USER %d", i+1),
				Brand:      pick(brands),
				PAN:        pan,
				CVV:        fmt.Sprintf("%03d", rand.Intn(1000)),
				PIN:        fmt.Sprintf("%04d", rand.Intn(10000)),
				Expiry:     fmt.Sprintf("%02d/%02d", rand.Intn(12)+1, 25+rand.Intn(5)),
				Tags:       randomTags(),
			})
			if err != nil {
				log.Printf("  WARN card %d: %v", i, err)
			} else {
				totalCreated++
			}
		}
		fmt.Printf("  + 5 cards\n")

		// 5 TOTP per vault
		for i := 0; i < 5; i++ {
			_, err := container.Secret.CreateTOTP(ctx, vaultID, secret.CreateTOTPInput{
				Title:   fmt.Sprintf("%s 2FA %d", pick(issuers), i+1),
				Issuer:  pick(issuers),
				Account: fmt.Sprintf("user%d@example.com", i),
				Secret:  randomBase32(16),
				Tags:    randomTags(),
			})
			if err != nil {
				log.Printf("  WARN totp %d: %v", i, err)
			} else {
				totalCreated++
			}
		}
		fmt.Printf("  + 5 TOTP\n")

		// 1 binary file per vault
		content := fmt.Sprintf("Binary test file for vault %s.\nRandom: %d\n", vaultName, rand.Int63())
		tmpFile, _ := os.CreateTemp("", "gk-seed-*.txt")
		_, _ = tmpFile.WriteString(content)
		_, _ = tmpFile.Seek(0, 0)
		_, ferr := container.Secret.CreateBinary(ctx, vaultID, secret.CreateBinaryInput{
			Title:    fmt.Sprintf("File %s %d", vaultName, vi+1),
			Filename: fmt.Sprintf("test_%s.txt", strings.ToLower(vaultName)),
			Data:     tmpFile,
			Size:     int64(len(content)),
			Tags:     []string{"test", "file"},
		})
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		if ferr != nil {
			log.Printf("  WARN file: %v", ferr)
		} else {
			totalCreated++
			fmt.Printf("  + 1 file\n")
		}
	}

	fmt.Println()
	fmt.Printf("=== Seed complete: %d vaults, %d secrets total ===\n", len(vaultNames), totalCreated)
	fmt.Println("Now open TUI and check sync!")
}

func pick(list []string) string {
	return list[rand.Intn(len(list))]
}

func randomTags() []string {
	all := []string{"work", "personal", "dev", "finance", "social", "backup", "important", "temp", "shared", "private"}
	n := rand.Intn(3) + 1
	tags := make([]string, 0, n)
	for i := 0; i < n; i++ {
		tags = append(tags, all[rand.Intn(len(all))])
	}
	return tags
}

func randomBase32(length int) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	b := make([]byte, length)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}
