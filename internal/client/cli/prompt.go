package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

var (
	readSecretFn = terminalReadSecret
	readLineFn   = terminalReadLine
)

func terminalReadLine(label string) (string, error) {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func terminalReadSecret(label string) ([]byte, error) {
	fmt.Print(label)
	secret, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func promptLine(label string) (string, error) {
	return readLineFn(label)
}

func promptSecret(label string) ([]byte, error) {
	return readSecretFn(label)
}

// maxSecretAttempts — сколько раз пользователь может повторить ввод секрета при несовпадении,
// прежде чем команда завершится с ошибкой. Ограничение защищает от зацикливания (например
// когда stdin не TTY).
const maxSecretAttempts = 3

// promptSecretConfirmed запрашивает секрет дважды и проверяет совпадение.
func promptSecretConfirmed(label, confirmLabel, mismatchMsg string) ([]byte, error) {
	for attempt := 0; attempt < maxSecretAttempts; attempt++ {
		first, err := promptSecret(label)
		if err != nil {
			return nil, err
		}
		second, err := promptSecret(confirmLabel)
		if err != nil {
			return nil, err
		}
		if bytesEqual(first, second) {
			return first, nil
		}
		fmt.Println(mismatchMsg)
	}
	return nil, errMismatch
}

// promptConfirm запрашивает подтверждение (y/N).
func promptConfirm(label string) (bool, error) {
	answer, err := promptLine(label + " [y/N]: ")
	if err != nil {
		return false, err
	}
	return isAffirmative(answer), nil
}

// isAffirmative распознаёт положительный ответ.
func isAffirmative(answer string) bool {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes", "н":
		return true
	default:
		return false
	}
}
