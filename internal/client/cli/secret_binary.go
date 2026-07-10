package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// FileCmd — группа команд для секретов типа binary (крупные файлы, хранятся в MinIO).
type FileCmd struct {
	Add      FileAddCmd      `cmd:"" help:"Upload a file as a binary secret (streamed + encrypted)."`
	List     FileListCmd     `cmd:"" help:"List binary secrets of a vault."`
	Download FileDownloadCmd `cmd:"" help:"Download and decrypt a binary secret to a local path."`
}

type FileAddCmd struct {
	Vault string `arg:"" help:"Vault name."`
	Path  string `arg:"" help:"Path to the local file to upload."`
}

func (c *FileAddCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	f, err := os.Open(c.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	title, err := promptLine(l.T("prompt_title"))
	if err != nil {
		return err
	}
	if title == "" {
		title = filepath.Base(c.Path)
	}
	note, err := promptLine(l.T("prompt_note"))
	if err != nil {
		return err
	}
	tagsRaw, err := promptLine(l.T("prompt_tags"))
	if err != nil {
		return err
	}

	if _, err := secret.CreateBinary(ctx, vaultID, secretuc.CreateBinaryInput{
		Title:    title,
		Filename: filepath.Base(c.Path),
		Data:     f,
		Size:     info.Size(),
		Note:     note,
		Tags:     parseTags(tagsRaw),
	}); err != nil {
		return err
	}
	fmt.Println(l.T("secret_created"))
	return nil
}

type FileListCmd struct {
	Vault string `arg:"" help:"Vault name."`
}

func (c *FileListCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	rows, err := secret.ListBinaryRows(ctx, vaultID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Println(l.T("secret_empty"))
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tFILENAME")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.ID, r.Row.Title, r.Row.Filename)
	}
	w.Flush()
	return nil
}

type FileDownloadCmd struct {
	Vault string `arg:"" help:"Vault name."`
	ID    string `arg:"" help:"Secret id (from 'secret file list')."`
	Out   string `arg:"" optional:"" help:"Output path. If a directory or omitted, uses the original filename."`
}

func (c *FileDownloadCmd) Run(auth *authuc.UseCase, vault *vaultuc.UseCase, secret *secretuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()
	if err := ensureUnlocked(ctx, auth, l); err != nil {
		return err
	}
	vaultID, err := openVaultByName(ctx, vault, c.Vault)
	if err != nil {
		return err
	}

	outPath := c.Out
	// Если выходной путь не задан или это существующая директория — подставляем оригинальное
	// имя файла из Tier 2a (BinaryRow.Filename), чтобы пользователю не нужно было помнить расширение.
	if outPath == "" || isDir(outPath) {
		rows, err := secret.ListBinaryRows(ctx, vaultID)
		if err != nil {
			return err
		}
		origName := ""
		for _, r := range rows {
			if r.ID == c.ID {
				origName = r.Row.Filename
				break
			}
		}
		if origName == "" {
			origName = c.ID // fallback: UUID, если filename не нашли
		}
		if outPath == "" {
			outPath = origName
		} else {
			outPath = filepath.Join(outPath, origName)
		}
	}

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := secret.DownloadBinary(ctx, vaultID, c.ID, out); err != nil {
		_ = os.Remove(outPath) // не оставляем частично записанный/повреждённый файл
		return err
	}
	fmt.Printf("%s %s\n", l.T("file_downloaded"), outPath)
	return nil
}

// isDir проверяет, является ли path существующей директорией.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
