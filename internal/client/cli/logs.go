package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
)

// LogsCmd — вывод клиентских логов из файла <data-dir>/client.log.
type LogsCmd struct {
	Lines int  `help:"Number of last lines to show (0 = all)." default:"50"`
	Clear bool `help:"Clear the log file." short:"c"`
}

// Run печатает/очищает содержимое клиентского лог-файла.
func (c *LogsCmd) Run(cfg *config.ClientConfig) error {
	logPath := filepath.Join(cfg.DataDir, "client.log")

	if c.Clear {
		if err := os.Truncate(logPath, 0); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Println("Log file cleared.")
		return nil
	}

	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		fmt.Println("No logs yet.")
		return nil
	}
	if err != nil {
		return err
	}

	if c.Lines == 0 {
		fmt.Print(string(data))
		return nil
	}

	lines := tailLines(string(data), c.Lines)
	fmt.Print(lines)
	return nil
}

// tailLines возвращает последние n строк текста.
func tailLines(s string, n int) string {
	end := len(s)
	count := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '\n' {
			count++
			if count > n {
				return s[i+1 : end]
			}
		}
	}
	return s[:end]
}
