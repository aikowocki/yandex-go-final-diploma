package secret_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
)

func TestErrConflict_Error(t *testing.T) {
	err := &secret.ErrConflict{Current: domain.Secret{ID: "s1", Version: 7}}
	assert.Contains(t, err.Error(), "7")
	assert.Contains(t, err.Error(), "conflict")
}
