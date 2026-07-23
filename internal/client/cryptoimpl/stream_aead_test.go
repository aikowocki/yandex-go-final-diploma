package cryptoimpl_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
)

func TestStreamAEAD_RoundTrip(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	ad := []byte("vault-1|secret-1")

	enc, err := cryptoimpl.NewStreamEncrypter(key, ad)
	require.NoError(t, err)

	chunks := [][]byte{
		[]byte("first chunk of the file..."),
		[]byte("second chunk..."),
		[]byte("final chunk."),
	}

	var ciphertexts [][]byte
	for i, c := range chunks {
		ct, err := enc.SealChunk(c, i == len(chunks)-1)
		require.NoError(t, err)
		ciphertexts = append(ciphertexts, ct)
	}

	dec, err := cryptoimpl.NewStreamDecrypter(key, ad, enc.StreamID())
	require.NoError(t, err)

	var got bytes.Buffer
	for i, ct := range ciphertexts {
		pt, err := dec.OpenChunk(ct, i == len(ciphertexts)-1)
		require.NoError(t, err)
		got.Write(pt)
	}

	var want bytes.Buffer
	for _, c := range chunks {
		want.Write(c)
	}
	assert.Equal(t, want.Bytes(), got.Bytes())
}

func TestStreamAEAD_WrongKeyFails(t *testing.T) {
	t.Parallel()

	enc, err := cryptoimpl.NewStreamEncrypter(mustKey(t), []byte("ad"))
	require.NoError(t, err)
	ct, err := enc.SealChunk([]byte("data"), true)
	require.NoError(t, err)

	dec, err := cryptoimpl.NewStreamDecrypter(mustKey(t), []byte("ad"), enc.StreamID())
	require.NoError(t, err)
	_, err = dec.OpenChunk(ct, true)
	require.Error(t, err)
}

func TestStreamAEAD_TamperedChunkFails(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	enc, err := cryptoimpl.NewStreamEncrypter(key, []byte("ad"))
	require.NoError(t, err)
	ct, err := enc.SealChunk([]byte("data"), true)
	require.NoError(t, err)
	ct[len(ct)-1] ^= 0xff

	dec, err := cryptoimpl.NewStreamDecrypter(key, []byte("ad"), enc.StreamID())
	require.NoError(t, err)
	_, err = dec.OpenChunk(ct, true)
	require.Error(t, err)
}

// TestStreamAEAD_ReorderedChunksFail проверяет, что переупорядочивание чанков (chunk1 подставлен
// как chunk0) обнаруживается — nonce/AD привязаны к позиции чанка в потоке.
func TestStreamAEAD_ReorderedChunksFail(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	enc, err := cryptoimpl.NewStreamEncrypter(key, []byte("ad"))
	require.NoError(t, err)
	ct0, err := enc.SealChunk([]byte("chunk-zero"), false)
	require.NoError(t, err)
	ct1, err := enc.SealChunk([]byte("chunk-one"), true)
	require.NoError(t, err)

	dec, err := cryptoimpl.NewStreamDecrypter(key, []byte("ad"), enc.StreamID())
	require.NoError(t, err)
	// Подставляем ct1 на место первого чанка — счётчик получателя (0) не совпадёт с тем, каким
	// был зашифрован ct1 (1), расшифровка должна провалиться.
	_, err = dec.OpenChunk(ct1, false)
	require.Error(t, err)

	_ = ct0
}

// TestStreamAEAD_TruncatedStreamDetected: если злоумышленник подсовывает чанк без last=true
// вместо реального последнего чанка (обрезание потока), получатель, ожидающий last=true на
// последнем чанке, должен получить ошибку расшифровки (AD не совпадёт).
func TestStreamAEAD_TruncatedStreamDetected(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	ad := []byte("ad")
	enc, err := cryptoimpl.NewStreamEncrypter(key, ad)
	require.NoError(t, err)
	// Зашифровано с last=false, хотя это последний чанк, который реально был отправлен.
	ct, err := enc.SealChunk([]byte("only chunk"), false)
	require.NoError(t, err)

	dec, err := cryptoimpl.NewStreamDecrypter(key, ad, enc.StreamID())
	require.NoError(t, err)
	// Получатель считает это последним чанком (транспорт завершился) и передаёт last=true.
	_, err = dec.OpenChunk(ct, true)
	require.Error(t, err, "AD mismatch (last flag) must be detected, not silently accepted")
}

func TestStreamAEAD_ExtraChunkAfterLastFails(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	ad := []byte("ad")
	enc, err := cryptoimpl.NewStreamEncrypter(key, ad)
	require.NoError(t, err)
	ct0, err := enc.SealChunk([]byte("last"), true)
	require.NoError(t, err)
	ct1, err := enc.SealChunk([]byte("extra after last"), false)
	require.NoError(t, err)

	dec, err := cryptoimpl.NewStreamDecrypter(key, ad, enc.StreamID())
	require.NoError(t, err)
	_, err = dec.OpenChunk(ct0, true)
	require.NoError(t, err)

	_, err = dec.OpenChunk(ct1, false)
	require.Error(t, err, "chunk received after the stream was marked finished must be rejected")
}

func TestStreamAEAD_DifferentStreamsProduceDifferentIDs(t *testing.T) {
	t.Parallel()

	key := mustKey(t)
	enc1, err := cryptoimpl.NewStreamEncrypter(key, []byte("ad"))
	require.NoError(t, err)
	enc2, err := cryptoimpl.NewStreamEncrypter(key, []byte("ad"))
	require.NoError(t, err)

	assert.NotEqual(t, enc1.StreamID(), enc2.StreamID())
}
