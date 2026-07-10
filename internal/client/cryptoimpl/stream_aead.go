package cryptoimpl

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// streamIDSize — длина случайного идентификатора потока в байтах.
const streamIDSize = 16

// chunkCounterSize — длина счётчика чанка в байтах (nonce = streamID || counter).
const chunkCounterSize = crypto.NonceSize - streamIDSize // 24 - 16 = 8

// StreamEncrypter шифрует поток чанков одного файла под одним VaultKey. НЕ безопасен для
// конкурентного использования из нескольких горутин.
type StreamEncrypter struct {
	key      []byte
	baseAD   []byte
	streamID [streamIDSize]byte
	counter  uint64
}

// NewStreamEncrypter создаёт шифратор потока со случайным streamID. baseAD — общий контекст
// (например vault_id|secret_id), к которому дополнительно привязывается счётчик и last-флаг
// каждого чанка.
func NewStreamEncrypter(key, baseAD []byte) (*StreamEncrypter, error) {
	if len(key) != crypto.KeySize {
		return nil, crypto.ErrInvalidKeySize
	}
	e := &StreamEncrypter{key: key, baseAD: baseAD}
	if _, err := rand.Read(e.streamID[:]); err != nil {
		return nil, fmt.Errorf("cryptoimpl: generate stream id: %w", err)
	}
	return e, nil
}

// StreamID возвращает случайный идентификатор потока — отправляется получателю один раз
// (первым сообщением/заголовком), чтобы он мог воспроизвести те же nonce на расшифровке.
func (e *StreamEncrypter) StreamID() []byte {
	id := make([]byte, streamIDSize)
	copy(id, e.streamID[:])
	return id
}

// SealChunk шифрует один чанк plaintext. last=true должен быть передан ровно для одного,
// последнего чанка потока (сигнал получателю, что поток корректно завершён).
func (e *StreamEncrypter) SealChunk(plaintext []byte, last bool) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(e.key)
	if err != nil {
		return nil, fmt.Errorf("cryptoimpl: init aead: %w", err)
	}

	nonce := chunkNonce(e.streamID, e.counter)
	ad := chunkAD(e.baseAD, e.counter, last)
	ciphertext := aead.Seal(nil, nonce, plaintext, ad)
	e.counter++
	return ciphertext, nil
}

// StreamDecrypter расшифровывает поток чанков, зашифрованный StreamEncrypter с тем же key/baseAD
// и переданным streamID.
type StreamDecrypter struct {
	key      []byte
	baseAD   []byte
	streamID [streamIDSize]byte
	counter  uint64
	done     bool // true после чанка с last=true — дальнейшие вызовы OpenChunk — ошибка
}

// NewStreamDecrypter создаёт дешифратор потока. streamID должен быть тем же, что вернул
// StreamEncrypter.StreamID() для этого потока.
func NewStreamDecrypter(key, baseAD, streamID []byte) (*StreamDecrypter, error) {
	if len(key) != crypto.KeySize {
		return nil, crypto.ErrInvalidKeySize
	}
	if len(streamID) != streamIDSize {
		return nil, fmt.Errorf("cryptoimpl: stream id must be %d bytes", streamIDSize)
	}
	d := &StreamDecrypter{key: key, baseAD: baseAD}
	copy(d.streamID[:], streamID)
	return d, nil
}

// OpenChunk расшифровывает один чанк. last должен точно соответствовать тому значению, с которым
// чанк был зашифрован (вызывающий код — grpcclient/blob.go — знает это из признака конца потока
// на транспортном уровне, например io.EOF на чтении файла). Вызов после чанка с last=true — ошибка
// (защита от обрезания/добавления данных после конца потока).
func (d *StreamDecrypter) OpenChunk(ciphertext []byte, last bool) ([]byte, error) {
	if d.done {
		return nil, fmt.Errorf("cryptoimpl: stream already finished, unexpected extra chunk")
	}

	aead, err := chacha20poly1305.NewX(d.key)
	if err != nil {
		return nil, fmt.Errorf("cryptoimpl: init aead: %w", err)
	}

	nonce := chunkNonce(d.streamID, d.counter)
	ad := chunkAD(d.baseAD, d.counter, last)
	plaintext, err := aead.Open(nil, nonce, ciphertext, ad)
	if err != nil {
		return nil, fmt.Errorf("cryptoimpl: decrypt chunk %d: %w", d.counter, err)
	}
	d.counter++
	if last {
		d.done = true
	}
	return plaintext, nil
}

// chunkNonce строит nonce = streamID || counter (big-endian), длина ровно crypto.NonceSize.
func chunkNonce(streamID [streamIDSize]byte, counter uint64) []byte {
	nonce := make([]byte, crypto.NonceSize)
	copy(nonce, streamID[:])
	binary.BigEndian.PutUint64(nonce[streamIDSize:], counter)
	return nonce
}

// chunkAD привязывает шифротекст чанка к baseAD + номеру чанка + признаку последнего чанка —
// подмена/переупорядочивание/обрезание чанков ломает расшифровку у получателя.
func chunkAD(baseAD []byte, counter uint64, last bool) []byte {
	ad := make([]byte, 0, len(baseAD)+chunkCounterSize+1)
	ad = append(ad, baseAD...)
	counterBytes := make([]byte, chunkCounterSize)
	binary.BigEndian.PutUint64(counterBytes, counter)
	ad = append(ad, counterBytes...)
	if last {
		ad = append(ad, 1)
	} else {
		ad = append(ad, 0)
	}
	return ad
}
