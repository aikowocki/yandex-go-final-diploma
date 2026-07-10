package secretcontent_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

func TestLoginPasswordTiers_EncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	c := cryptoimpl.Crypto{}
	vaultKey := make([]byte, crypto.KeySize)
	_, err := rand.Read(vaultKey)
	require.NoError(t, err)

	row := secretcontent.LoginPasswordRow{
		V:        secretcontent.LoginPasswordSchemaV1,
		Title:    "GitHub",
		Tags:     []string{"work", "dev"},
		URI:      "https://github.com",
		Username: "alice",
	}
	index := secretcontent.LoginPasswordIndex{
		V:            secretcontent.LoginPasswordSchemaV1,
		Note:         "personal account",
		CustomFields: []secretcontent.KeyValue{{Key: "recovery", Value: "codes"}},
	}
	payload := secretcontent.LoginPasswordPayload{
		V:        secretcontent.LoginPasswordSchemaV1,
		Password: "hunter2hunter2",
	}

	encRow, err := c.EncryptStruct(vaultKey, nil, row)
	require.NoError(t, err)
	encIndex, err := c.EncryptStruct(vaultKey, nil, index)
	require.NoError(t, err)
	encPayload, err := c.EncryptStruct(vaultKey, nil, payload)
	require.NoError(t, err)

	var gotRow secretcontent.LoginPasswordRow
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encRow, &gotRow))
	assert.Equal(t, row, gotRow)

	var gotIndex secretcontent.LoginPasswordIndex
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encIndex, &gotIndex))
	assert.Equal(t, index, gotIndex)

	var gotPayload secretcontent.LoginPasswordPayload
	require.NoError(t, c.DecryptStruct(vaultKey, nil, encPayload, &gotPayload))
	assert.Equal(t, payload, gotPayload)
}
