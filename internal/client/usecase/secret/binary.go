package secret

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// binaryStreamChunkSize — размер чанка потокового AEAD при шифровании файла на диске Компромисс память/накладные расходы AEAD-тега на чанк.
const binaryStreamChunkSize = 256 * 1024

type CreateBinaryInput struct {
	Title        string
	Tags         []string
	Filename     string
	Mime         string
	Note         string
	CustomFields []secretcontent.KeyValue
	Data         io.Reader
	Size         int64 // -1, если неизвестен
	OTPCodes     []secretcontent.OTPCode
}

func (in CreateBinaryInput) toRow() secretcontent.BinaryRow {
	return secretcontent.BinaryRow{V: secretcontent.BinarySchemaV1, Title: in.Title, Tags: in.Tags, Filename: in.Filename}
}

func (in CreateBinaryInput) toIndex(size int64) secretcontent.BinaryIndex {
	return secretcontent.BinaryIndex{
		V: secretcontent.BinarySchemaV1, Size: size, Mime: in.Mime, Note: in.Note, CustomFields: in.CustomFields,
	}
}

func (in CreateBinaryInput) toPayload() secretcontent.BinaryPayload {
	return secretcontent.BinaryPayload{V: secretcontent.BinarySchemaV1, OTPCodes: in.OTPCodes}
}

func (u *UseCase) CreateBinary(ctx context.Context, vaultID string, input CreateBinaryInput) (string, error) {
	if input.Title == "" {
		return "", ErrEmptyTitle
	}
	if input.Data == nil {
		return "", ErrEmptyBinaryData
	}

	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return "", err
	}

	row, index, payload := input.toRow(), input.toIndex(input.Size), input.toPayload()

	secretID, err := createTyped(ctx, u, vaultID, int32(domain.SecretTypeBinary), row, index, payload)
	if err != nil {
		return "", err
	}

	blobRef, blobSize, err := u.uploadBinaryData(ctx, token, vaultKey, vaultID, secretID, input.Data)
	if err != nil {
		return secretID, fmt.Errorf("secret created but blob upload failed (secret_id=%s): %w", secretID, err)
	}

	if _, err := u.server.AttachBlob(ctx, token, secretID, createVersion, blobRef, blobSize); err != nil {
		return secretID, fmt.Errorf("secret created, blob uploaded, but AttachBlob failed (secret_id=%s, blob_ref=%s): %w", secretID, blobRef, err)
	}
	return secretID, nil
}

// uploadBinaryData шифрует Data потоково (StreamEncrypter, чанки binaryStreamChunkSize) и
// стримит результат на сервер. AAD чанков привязана к тому же контексту, что и обычные тиры
// секрета (vault|secret|version|payload) — переиспользование secretAAD.
func (u *UseCase) uploadBinaryData(ctx context.Context, token string, vaultKey []byte, vaultID, secretID string, data io.Reader) (blobRef string, blobSize int64, err error) {
	ad := secretAAD(vaultID, secretID, createVersion, "blob")
	enc, err := cryptoimpl.NewStreamEncrypter(vaultKey, ad)
	if err != nil {
		return "", 0, fmt.Errorf("init stream encrypter: %w", err)
	}

	pr, pw := io.Pipe()
	encodeErrCh := make(chan error, 1)
	go func() {
		encodeErrCh <- encodeStream(enc, data, pw)
	}()

	blobRef, blobSize, err = u.server.UploadBlob(ctx, token, secretID, pr)
	if encErr := <-encodeErrCh; encErr != nil && err == nil {
		err = encErr
	}
	if err != nil {
		return "", 0, err
	}
	return blobRef, blobSize, nil
}

func encodeStream(enc *cryptoimpl.StreamEncrypter, data io.Reader, w io.WriteCloser) error {
	defer w.Close()

	if err := writeFrame(w, enc.StreamID()); err != nil {
		return err
	}

	buf := make([]byte, binaryStreamChunkSize)
	n, err := data.Read(buf)
	pending := append([]byte(nil), buf[:n]...)
	pendingErr := err

	for {
		if pendingErr != nil && !errors.Is(pendingErr, io.EOF) {
			return fmt.Errorf("read data: %w", pendingErr)
		}

		nextN, nextErr := 0, io.EOF
		if errors.Is(pendingErr, io.EOF) {
			// pending — последний чанк (может быть len==0, если файл пустой — всё равно
			// нужно отправить один last-чанк, чтобы получатель знал, что поток корректно закрыт).
		} else {
			nextN, nextErr = data.Read(buf)
		}

		last := errors.Is(pendingErr, io.EOF)
		ct, eerr := enc.SealChunk(pending, last)
		if eerr != nil {
			return fmt.Errorf("encrypt chunk: %w", eerr)
		}
		if werr := writeFrame(w, ct); werr != nil {
			return werr
		}
		if last {
			return nil
		}

		pending = append([]byte(nil), buf[:nextN]...)
		pendingErr = nextErr
	}
}

func (u *UseCase) DownloadBinary(ctx context.Context, vaultID, secretID string, w io.Writer) error {
	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return err
	}

	local, ok, err := u.local.GetSecret(ctx, secretID)
	if err != nil {
		return err
	}
	version := createVersion
	if ok {
		version = local.Version
	}

	rc, err := u.server.DownloadBlob(ctx, token, secretID)
	if err != nil {
		return err
	}
	defer rc.Close()

	ad := secretAAD(vaultID, secretID, version, "blob")
	return decodeStream(vaultKey, ad, rc, w)
}

func (u *UseCase) ListBinaryRows(ctx context.Context, vaultID string) ([]TypedRow[secretcontent.BinaryRow], error) {
	return listRowsTyped[secretcontent.BinaryRow](ctx, u, vaultID, int32(domain.SecretTypeBinary))
}

func (u *UseCase) GetBinaryDetail(ctx context.Context, vaultID, secretID string) (TypedDetail[secretcontent.BinaryRow, secretcontent.BinaryIndex, secretcontent.BinaryPayload], error) {
	return getDetailTyped[secretcontent.BinaryRow, secretcontent.BinaryIndex, secretcontent.BinaryPayload](ctx, u, vaultID, secretID)
}

func decodeStream(key, ad []byte, r io.Reader, w io.Writer) error {
	streamID, err := readFrame(r)
	if err != nil {
		return fmt.Errorf("read stream id: %w", err)
	}
	dec, err := cryptoimpl.NewStreamDecrypter(key, ad, streamID)
	if err != nil {
		return fmt.Errorf("init stream decrypter: %w", err)
	}

	pending, err := readFrame(r)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("empty blob stream: %w", ErrEmptyBinaryData)
		}
		return fmt.Errorf("read chunk: %w", err)
	}

	for {
		next, nextErr := readFrame(r)
		last := errors.Is(nextErr, io.EOF)
		if nextErr != nil && !last {
			return fmt.Errorf("read chunk: %w", nextErr)
		}

		pt, derr := dec.OpenChunk(pending, last)
		if derr != nil {
			return fmt.Errorf("decrypt chunk: %w", derr)
		}
		if _, werr := w.Write(pt); werr != nil {
			return fmt.Errorf("write output: %w", werr)
		}
		if last {
			return nil
		}
		pending = next
	}
}
