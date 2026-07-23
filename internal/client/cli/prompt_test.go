package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type secretStep struct {
	val []byte
	err error
}

// scriptSecrets подменяет readSecretFn последовательностью заранее заданных ответов.
func scriptSecrets(t *testing.T, steps ...secretStep) {
	t.Helper()
	orig := readSecretFn
	t.Cleanup(func() { readSecretFn = orig })

	i := 0
	readSecretFn = func(string) ([]byte, error) {
		if i >= len(steps) {
			t.Fatalf("unexpected extra secret read (only %d scripted)", len(steps))
		}
		s := steps[i]
		i++
		return s.val, s.err
	}
}

func TestPromptSecretConfirmed_MatchFirstTry(t *testing.T) {
	scriptSecrets(t,
		secretStep{val: []byte("pw")},
		secretStep{val: []byte("pw")},
	)

	got, err := promptSecretConfirmed("l", "c", "mismatch")
	require.NoError(t, err)
	assert.Equal(t, []byte("pw"), got)
}

func TestPromptSecretConfirmed_RetriesThenMatches(t *testing.T) {
	scriptSecrets(t,
		secretStep{val: []byte("pw")},
		secretStep{val: []byte("typo")}, // первая пара не совпала
		secretStep{val: []byte("pw")},
		secretStep{val: []byte("pw")}, // вторая пара совпала
	)

	got, err := promptSecretConfirmed("l", "c", "mismatch")
	require.NoError(t, err)
	assert.Equal(t, []byte("pw"), got)
}

func TestPromptSecretConfirmed_AllAttemptsMismatch(t *testing.T) {
	// maxSecretAttempts=3 → 3 пары несовпадений (6 чтений).
	scriptSecrets(t,
		secretStep{val: []byte("a")}, secretStep{val: []byte("b")},
		secretStep{val: []byte("a")}, secretStep{val: []byte("b")},
		secretStep{val: []byte("a")}, secretStep{val: []byte("b")},
	)

	_, err := promptSecretConfirmed("l", "c", "mismatch")
	assert.ErrorIs(t, err, errMismatch)
}

func TestPromptSecretConfirmed_ReadErrorReturnsImmediately(t *testing.T) {
	boom := errors.New("stdin closed")
	scriptSecrets(t, secretStep{err: boom})

	_, err := promptSecretConfirmed("l", "c", "mismatch")
	assert.ErrorIs(t, err, boom, "read error must be returned immediately, not retried")
}

func TestIsAffirmative(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"y", true},
		{"Y", true},
		{"yes", true},
		{"YES", true},
		{"н", true},   // клавиша y в русской раскладке
		{"Н", true},   // Shift+y в русской раскладке
		{" y ", true}, // пробелы обрезаются
		{"n", false},
		{"N", false},
		{"т", false},  // клавиша n в русской раскладке → отказ
		{"да", false}, // слова осознанно не принимаем
		{"нет", false},
		{"", false}, // дефолт — No
		{"maybe", false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, isAffirmative(tt.in))
		})
	}
}
