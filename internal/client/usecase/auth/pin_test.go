package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

// TestPIN_RoundTrip: установка PIN на тёплой сессии, мягкая блокировка (SoftLock стирает
// MasterKey, но оставляет PIN-материал), затем разблокировка тем же PIN восстанавливает сессию.
func TestPIN_RoundTrip(t *testing.T) {
	sess := session.New()
	uc := authuc.New(nil, cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, nil, sess, nil)

	// Эмулируем разблокированную сессию: кладём произвольный 32-байтный MasterKey.
	mk := make([]byte, 32)
	for i := range mk {
		mk[i] = byte(i + 1)
	}
	// Копия для сравнения: SoftLock затирает исходный слайс на месте (это ожидаемо для
	// безопасности — SetMasterKey хранит ссылку, а не копию).
	expected := append([]byte(nil), mk...)
	sess.SetMasterKey(mk)

	require.False(t, uc.HasPIN())
	require.NoError(t, uc.SetPIN([]byte("1234")))
	require.True(t, uc.HasPIN())

	// Мягкая блокировка (авто-лок): MasterKey стёрт, PIN-материал сохранён.
	sess.SoftLock()
	require.False(t, sess.Unlocked())
	require.True(t, sess.HasPIN())

	// Верный PIN восстанавливает MasterKey.
	require.NoError(t, uc.UnlockWithPIN([]byte("1234")))
	require.True(t, sess.Unlocked())
	got, ok := sess.MasterKey()
	require.True(t, ok)
	assert.Equal(t, expected, got)
}

// TestPIN_WrongPIN: неверный PIN не разблокирует (ошибка расшифровки обёртки).
func TestPIN_WrongPIN(t *testing.T) {
	sess := session.New()
	uc := authuc.New(nil, cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, nil, sess, nil)
	sess.SetMasterKey(make([]byte, 32))

	require.NoError(t, uc.SetPIN([]byte("1234")))
	sess.SoftLock()

	err := uc.UnlockWithPIN([]byte("9999"))
	require.Error(t, err)
	assert.False(t, sess.Unlocked())
}

// TestPIN_TooShort: слишком короткий PIN отклоняется.
func TestPIN_TooShort(t *testing.T) {
	sess := session.New()
	uc := authuc.New(nil, cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, nil, sess, nil)
	sess.SetMasterKey(make([]byte, 32))
	require.Error(t, uc.SetPIN([]byte("12")))
}

// TestPIN_FullLockClearsPIN: полная блокировка (Lock) стирает PIN-материал — нужен master-пароль.
func TestPIN_FullLockClearsPIN(t *testing.T) {
	sess := session.New()
	uc := authuc.New(nil, cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, nil, sess, nil)
	sess.SetMasterKey(make([]byte, 32))
	require.NoError(t, uc.SetPIN([]byte("1234")))

	sess.Lock()
	require.False(t, sess.HasPIN())
	require.ErrorIs(t, uc.UnlockWithPIN([]byte("1234")), authuc.ErrPINNotSet)
}
