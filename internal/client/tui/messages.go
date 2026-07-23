package tui

import (
	"time"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// --- Навигационные сообщения (переключение экранов) ---

// switchScreenMsg переключает root model на указанный экран, опционально перенося контекст
// (выбранный vault, редактируемый secret, конфликт) в соответствующую дочернюю модель.
type switchScreenMsg struct {
	screen screenID

	// Контекст для screenDashboard (возврат после формы/конфликта).
	vaultID   string
	vaultName string

	// Контекст для screenForm. editSecret=true → редактирование существующего секрета.
	editSecret    bool
	secretID      string
	secretVer     int64
	editType      domain.SecretType // тип редактируемого секрета
	editData      formData          // загруженные поля секрета для редактирования
	preselectType domain.SecretType // предвыбранный тип при создании (0 = login/password фолбэк)
	// openInEditMode — открыть существующий секрет сразу в edit mode (шорткат [e] в таблице),
	// а не в обычном read-only view mode (Enter/[v]).
	openInEditMode bool

	// Контекст для screenConflict. Единый тип конфликта для всех типов секретов
	conflict *secret.GenericConflict
}

// --- Auth ---

type loginSuccessMsg struct{}
type registerSuccessMsg struct{}
type loginErrMsg struct{ err error }

// --- Unlock ---

type unlockSuccessMsg struct{}
type unlockErrMsg struct{ err error }

// recoveryCodesGeneratedMsg — recovery codes сгенерированы после SetupEncryption.
type recoveryCodesGeneratedMsg struct{ codes []string }

// pinSetMsg — PIN успешно установлен на экране разблокировки.
type pinSetMsg struct{}

// --- Vault ---

type vaultsLoadedMsg struct{ vaults []vaultuc.DecryptedVault }
type vaultsErrMsg struct{ err error }
type vaultCreatedMsg struct{ id string }

// --- Secrets (dashboard table) ---

type rowsLoadedMsg struct{ rows []secret.SummaryRow }
type rowsErrMsg struct{ err error }
type payloadRevealedMsg struct {
	secretID string
	password string
}
type payloadRevealErrMsg struct{ err error }

// cardRevealedMsg — раскрытые по [R] чувствительные поля карты.
type cardRevealedMsg struct {
	secretID string
	cvv      string
	pin      string
	pan      string
}

// totpAllCodesMsg — все TOTP-коды текущей папки (режим totp_reveal_mode=all).
type totpAllCodesMsg struct{ codes map[string]string }

// openDownloadPickerMsg — клик по кликабельной ячейке «Скачать» binary-секрета в таблице
// dashboard: запускает флоу скачивания с выбором папки назначения.
type openDownloadPickerMsg struct {
	secretID string
	filename string
}

// --- Sync ---

type syncDoneMsg struct{}
type syncErrMsg struct{ err error }
type syncScopeConfirmedMsg struct{}

// backgroundSyncTickMsg — тик таймера фонового sync.
// settingsSyncToggledMsg — флаг синхронизации vault переключён в Settings (успех).
type settingsSyncToggledMsg struct{}

// settingsSavedMsg — конфиг успешно сохранён на диск после изменения настройки.
type settingsSavedMsg struct{}

type backgroundSyncTickMsg struct{}

// outboxConflictCountMsg — обновление счётчика нерешённых outbox-конфликтов для бэджа в топ-баре.
type outboxConflictCountMsg struct{ count int }

// backgroundSyncDoneMsg — результат тихого фонового sync (ошибка молча логируется, не
// показывается пользователю тостом — сеть недоступна — это ожидаемое, а не аварийное событие).
type backgroundSyncDoneMsg struct{ err error }

// --- Conflict ---

// conflictMsg сигнализирует о конфликте версий, обнаруженном при сохранении секрета.
type conflictMsg struct{ conflict *secret.GenericConflict }
type conflictResolvedMsg struct{}
type conflictErrMsg struct{ err error }

// --- TOTP ---

type totpTickMsg struct{ t time.Time }

// --- Notify ---

type toastMsg struct{ text string }
type toastExpiredMsg struct{}

// --- Autolock ---

type autolockMsg struct{}

// autolockChangedMsg — таймаут автоблокировки изменён в Settings; App обновляет своё поле.
type autolockChangedMsg struct{ timeout time.Duration }

// --- Cell copy feedback ---

// copiedCellExpiredMsg сбрасывает визуальный индикатор ✓ в ячейке таблицы.
type copiedCellExpiredMsg struct{}
