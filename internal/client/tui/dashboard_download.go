package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
)

// downloadPickerPopup — обёртка над bubbles/filepicker, настроенная на выбор директории
// (DirAllowed=true, FileAllowed=false), в которую будет сохранён расшифрованный файл.
type downloadPickerPopup struct {
	fp filepicker.Model

	vaultID  string
	secretID string
	filename string // Оригинальное имя файла секрета (Subtitle из SummaryRow)
}

// newDownloadPickerPopup инициализирует попап с курсором в домашней директории пользователя.
func newDownloadPickerPopup(vaultID, secretID, filename string) downloadPickerPopup {
	home, _ := os.UserHomeDir()
	fp := filepicker.New()
	fp.CurrentDirectory = home
	fp.ShowHidden = false
	fp.DirAllowed = true
	fp.FileAllowed = false
	fp.SetHeight(15)
	if filename == "" {
		filename = "download"
	}
	return downloadPickerPopup{fp: fp, vaultID: vaultID, secretID: secretID, filename: filename}
}

func (m downloadPickerPopup) Init() tea.Cmd {
	return m.fp.Init()
}

// update обрабатывает ввод попапа. done=true означает, что попап нужно закрыть (папка
// выбрана — скачивание запущено в фоне, либо пользователь отменил по Esc).
func (m downloadPickerPopup) update(ctx context.Context, container *app.Container, msg tea.Msg) (downloadPickerPopup, tea.Cmd, bool) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyEsc {
		return m, nil, true
	}

	var cmd tea.Cmd
	m.fp, cmd = m.fp.Update(msg)

	if didSelect, dir := m.fp.DidSelectFile(msg); didSelect {
		return m, m.startDownload(ctx, container, dir), true
	}
	return m, cmd, false
}

// startDownload скачивает и расшифровывает binary-секрет в выбранную директорию.
func (m downloadPickerPopup) startDownload(ctx context.Context, container *app.Container, dir string) tea.Cmd {
	vaultID := m.vaultID
	secretID := m.secretID
	outPath := filepath.Join(dir, filepath.Base(m.filename))
	return func() tea.Msg {
		// Toast (не loginErrMsg) для ошибки: попап скачивания закрывается сразу после выбора
		// папки, поэтому к моменту завершения этой команды dashboard уже не
		// в фокусе focusDownloadPicker — сообщение должно доходить независимо от текущего
		// экрана/фокуса, а toastModel.update обрабатывается на каждый msg в App.Update.
		f, err := os.Create(outPath)
		if err != nil {
			return toastMsg{text: fmt.Sprintf("✗ %v", fmt.Errorf("create file: %w", err))}
		}
		defer func() { _ = f.Close() }()

		if err := container.Secret.DownloadBinary(ctx, vaultID, secretID, f); err != nil {
			_ = os.Remove(outPath)
			return toastMsg{text: fmt.Sprintf("✗ %v", fmt.Errorf("download: %w", err))}
		}
		return toastMsg{text: fmt.Sprintf(container.Localizer.T("tui_toast_downloaded"), outPath)}
	}
}

func (m downloadPickerPopup) view(l localizerT) string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("📂 " + l.T("tui_download_picker_title")))
	b.WriteString("\n\n")
	b.WriteString(m.fp.View())
	b.WriteString("\n\n")
	b.WriteString(styles.HelpText.Render(l.T("tui_help_download_picker")))
	return b.String()
}
