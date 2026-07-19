package tui

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
	syncuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/sync"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// ansiEscapeRe — паттерн ANSI SGR-последовательностей для вычисления видимой ширины строки.
var ansiEscapeRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// typeTab — одна вкладка второго ряда (тип секрета). all=0 означает «без фильтра».
type typeTab struct {
	label      string
	secretType domain.SecretType // 0 = All
}

func typeTabs(l localizerT) []typeTab {
	return []typeTab{
		{label: "📋 " + l.T("tui_tab_all"), secretType: 0},
		{label: "🔑 " + l.T("tui_tab_logins"), secretType: domain.SecretTypeLoginPassword},
		{label: "💳 " + l.T("tui_tab_cards"), secretType: domain.SecretTypeBankCard},
		{label: "🔢 " + l.T("tui_tab_totp"), secretType: domain.SecretTypeTOTP},
		{label: "📁 " + l.T("tui_tab_files"), secretType: domain.SecretTypeBinary},
		{label: "📝 " + l.T("tui_tab_notes"), secretType: domain.SecretTypeText},
	}
}

type localizerT interface {
	T(id string) string
}

// dashboardFocus определяет, какая часть dashboard сейчас принимает фокус ввода.
type dashboardFocus int

const (
	focusTable   dashboardFocus = iota // Таблица секретов (навигация ↑↓, шорткаты)
	focusCommand                       // Нижняя командная/поисковая строка
	focusVaultCreate
	focusSettings
	focusUser
	focusSyncScope      // Попап выбора папок для синхронизации (первый вход)
	focusDetail         // read-only карточка просмотра секрета [V]
	focusLogs           // Попап просмотра логов (:logs)
	focusDownloadPicker // Попап выбора папки для скачивания binary-секрета
)

type dashboardModel struct {
	ctx       context.Context
	container *app.Container
	l         localizerT

	vaults      []vaultuc.DecryptedVault
	vaultCursor int
	vaultsErr   error
	syncing     bool

	typeCursor int // индекс в typeTabs()

	table dashboardTableModel

	focus          dashboardFocus
	commandLine    commandLineModel
	vaultCreate    vaultCreateModel
	settings       settingsPopup
	userMenu       userMenuPopup
	syncScope      syncScopePopup
	logs           logsPopup
	downloadPicker downloadPickerPopup

	err error
	// outboxConflictCount — количество нерешённых outbox-конфликтов (кэш для отображения бэджа
	// в топ-баре без синхронного похода в БД на каждый рендер). Обновляется после каждого
	// фонового sync (backgroundSyncDoneMsg) и после разрешения конфликта.
	outboxConflictCount int
	// lastBackgroundSyncErr — ошибка последнего фонового sync (обычно офлайн/сеть недоступна).
	// Показывается тихой пометкой в топ-баре, без toast — фоновый sync не должен прерывать
	// работу пользователя каждые backgroundSyncInterval при отсутствии сети.
	lastBackgroundSyncErr error
	// lastSyncOK — время последнего успешного фонового sync (для отображения «synced N ago»).
	lastSyncOK time.Time
	// backgroundSyncing — фоновый sync сейчас выполняется (индикатор ↻ в топ-баре).
	backgroundSyncing bool
	// syncProgressLabel — текущий этап sync для отображения в топ-баре (real-time).
	syncProgressLabel string
	// initialLoadDone — первая загрузка папок в этой сессии уже прошла.
	initialLoadDone bool
}

func newDashboardModel(ctx context.Context, container *app.Container) dashboardModel {
	return dashboardModel{
		ctx:         ctx,
		container:   container,
		l:           container.Localizer,
		table:       newDashboardTableModel(ctx, container),
		commandLine: newCommandLineModel(),
		vaultCreate: newVaultCreateModel(),
	}
}

// backgroundSyncInterval — период фоновой синхронизации в dashboard. Не блокирует UI: Sync()
// выполняется в отдельной tea.Cmd, а таблица продолжает реагировать на ввод как обычно.
const backgroundSyncInterval = 30 * time.Second

func (m dashboardModel) Init() tea.Cmd {
	// Первая загрузка: папки грузим с сервера.
	// Sync запускается ТОЛЬКО если пользователь уже выбрал scope (SyncScopeChosen).
	// Если scope ещё не выбран — sync стартует после подтверждения выбора (syncScopeConfirmedMsg).
	cmds := []tea.Cmd{m.loadVaultsFromServer(), m.backgroundSyncTick(), m.refreshOutboxConflictCount()}
	if m.container.Sync.SyncScopeChosen(m.ctx) {
		cmds = append(cmds, m.launchBackgroundSync())
	}
	return tea.Batch(cmds...)
}

// refreshOutboxConflictCount пересчитывает количество нерешённых outbox-конфликтов для
// бэджа в топ-баре.
func (m dashboardModel) refreshOutboxConflictCount() tea.Cmd {
	container := m.container
	ctx := m.ctx
	return func() tea.Msg {
		entries, err := container.Secret.ListOutboxConflicts(ctx)
		if err != nil {
			return outboxConflictCountMsg{count: 0}
		}
		return outboxConflictCountMsg{count: len(entries)}
	}
}

// backgroundSyncTick планирует следующий тихий фоновый sync через backgroundSyncInterval.
func (m dashboardModel) backgroundSyncTick() tea.Cmd {
	return tea.Tick(backgroundSyncInterval, func(time.Time) tea.Msg {
		return backgroundSyncTickMsg{}
	})
}

// Refresh перезагружает данные текущего vault/type. На первом входе в dashboard (сразу после
// login/unlock, initialLoadDone==false) грузит папки с сервера — иначе на новом устройстве с
// пустым локальным кешем список был бы пуст и попап выбора синка не появился бы. При возврате
// из формы/конфликта (initialLoadDone==true) — быстрый локальный рефреш.
func (m dashboardModel) Refresh() tea.Cmd {
	if !m.initialLoadDone {
		cmds := []tea.Cmd{m.loadVaultsFromServer(), m.backgroundSyncTick(), m.refreshOutboxConflictCount()}
		if m.container.Sync.SyncScopeChosen(m.ctx) {
			cmds = append(cmds, m.launchBackgroundSync())
		}
		return tea.Batch(cmds...)
	}
	return tea.Batch(m.loadVaults(), m.table.reload(), m.refreshOutboxConflictCount())
}

func (m dashboardModel) currentVault() (vaultuc.DecryptedVault, bool) {
	if len(m.vaults) == 0 || m.vaultCursor >= len(m.vaults) {
		return vaultuc.DecryptedVault{}, false
	}
	return m.vaults[m.vaultCursor], true
}

func (m dashboardModel) currentTypeTab() typeTab {
	tabs := typeTabs(m.l)
	if m.typeCursor >= len(tabs) {
		return tabs[0]
	}
	return tabs[m.typeCursor]
}

func (m dashboardModel) view(width, height int) string {
	var b strings.Builder

	b.WriteString(m.renderTopBar(width))
	b.WriteString("\n")
	b.WriteString(m.renderVaultTabs())
	b.WriteString("\n")
	b.WriteString(m.renderTypeTabs())
	b.WriteString("\n\n")

	switch m.focus {
	case focusSettings:
		b.WriteString(m.settings.view(m.l))
	case focusUser:
		b.WriteString(m.userMenu.view(m.ctx, m.container, m.l))
	case focusVaultCreate:
		b.WriteString(m.vaultCreate.view(m.l))
	case focusSyncScope:
		b.WriteString(m.syncScope.view(m.l))
	case focusLogs:
		b.WriteString(m.logs.view())
	case focusDownloadPicker:
		b.WriteString(m.downloadPicker.view(m.l))
	default:
		if m.vaultsErr != nil {
			b.WriteString(styles.ErrorText.Render(m.vaultsErr.Error()))
			b.WriteString("\n")
		} else if len(m.vaults) == 0 {
			b.WriteString(styles.Subtitle.Render(m.l.T("tui_no_vaults")))
		} else {
			// Доступная высота = total - шапка(5 строк: topbar+vaults+types+пустая+заголовок) - футер(3: help+cmdline+пустая).
			tableHeight := height - 11
			if tableHeight < 5 {
				tableHeight = 5
			}
			b.WriteString(m.table.view(width, tableHeight))
		}
	}

	// Строка помощи (только в обычном режиме таблицы — попапы показывают свои подсказки).
	if m.focus == focusTable {
		b.WriteString("\n")
		b.WriteString(styles.HelpText.Render(m.l.T("tui_help_dashboard")))
	}

	b.WriteString("\n")
	b.WriteString(m.commandLine.view(m.l, m.focus == focusCommand))
	return b.String()
}

func (m dashboardModel) update(msg tea.Msg) (dashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease {
			dm, cmd := m.handleTabClick(msg)
			if cmd != nil {
				return dm, cmd
			}
			// Клик не попал в таб — делегируем таблице (click-to-copy).
			if m.focus == focusTable {
				var tCmd tea.Cmd
				dm.table, tCmd = dm.table.update(msg)
				return dm, tCmd
			}
			return dm, nil
		}
		// Scroll колёсиком в попапе логов.
		if m.focus == focusLogs && (msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
			m.logs = m.logs.updateMouse(msg)
			return m, nil
		}

	case vaultsLoadedMsg:
		m.vaultsErr = nil
		// Сохраняем ID текущей папки до обновления списка.
		prevID := m.currentVaultID()
		m.vaults = msg.vaults
		// Восстанавливаем курсор на тот же vault по ID (порядок мог измениться после сортировки).
		m = m.selectVaultByID(prevID)
		if m.vaultCursor >= len(m.vaults) && len(m.vaults) > 0 {
			m.vaultCursor = len(m.vaults) - 1
		}
		firstLoad := !m.initialLoadDone
		m.initialLoadDone = true
		// Попап выбора синхронизации — только на ПЕРВОЙ загрузке сессии и только если папки
		// уже существуют. После ручного создания папки попап не показывается — пользователь явно хочет его синкать.
		if firstLoad && len(m.vaults) > 0 && m.focus == focusTable && !m.container.Sync.SyncScopeChosen(m.ctx) {
			m.focus = focusSyncScope
			m.syncScope = newSyncScopePopup(m.vaults)
			return m, nil
		}
		// Тост при обнаружении новых несинхронизированных папок (не первая загрузка).
		if !firstLoad {
			newUnsyncedCount := 0
			for _, v := range m.vaults {
				if !v.SyncEnabled {
					newUnsyncedCount++
				}
			}
			if newUnsyncedCount > 0 {
				var cmd tea.Cmd
				m.table, cmd = m.table.setVaultAndType(m.currentVaultID(), m.currentTypeTab().secretType)
				return m, tea.Batch(
					cmd,
					showToast(m.l.T("tui_toast_new_vaults_found")),
				)
			}
		}
		var cmd tea.Cmd
		m.table, cmd = m.table.setVaultAndType(m.currentVaultID(), m.currentTypeTab().secretType)
		return m, cmd

	case vaultsErrMsg:
		m.vaultsErr = msg.err
		return m, nil

	case vaultCreatedMsg:
		m.focus = focusTable
		return m, m.loadVaults()

	case syncDoneMsg:
		m.syncing = false
		return m, tea.Batch(m.loadVaults(), showToast(m.l.T("tui_toast_synced")))

	case syncErrMsg:
		m.syncing = false
		m.err = msg.err
		return m, nil

	case syncScopeConfirmedMsg:
		m.focus = focusTable
		// Scope выбран — запускаем первый sync.
		m.backgroundSyncing = true
		var cmd tea.Cmd
		m.table, cmd = m.table.setVaultAndType(m.currentVaultID(), m.currentTypeTab().secretType)
		return m, tea.Batch(
			cmd,
			m.launchBackgroundSync(),
		)

	case backgroundSyncTickMsg:
		// Не запускаем новый sync если предыдущий ещё выполняется.
		if m.backgroundSyncing {
			return m, m.backgroundSyncTick() // просто перепланируем tick
		}
		m.backgroundSyncing = true
		m.syncProgressLabel = ""
		return m, m.launchBackgroundSync()

	case syncProgressWithNext:
		m.syncProgressLabel = msg.label
		// Планируем следующее чтение из канала прогресса.
		return m, waitForSyncProgress(msg.progressCh, msg.errCh)

	case backgroundSyncDoneMsg:
		m.backgroundSyncing = false
		m.syncProgressLabel = ""
		m.lastBackgroundSyncErr = msg.err
		if msg.err == nil {
			m.lastSyncOK = time.Now()
			// Перезагружаем папки с сервера и планируем следующий sync.
			return m, tea.Batch(m.table.reload(), m.loadVaultsFromServer(), m.refreshOutboxConflictCount(), m.backgroundSyncTick())
		}
		// Ошибка — планируем повтор через 30с.
		return m, tea.Batch(m.refreshOutboxConflictCount(), m.backgroundSyncTick())

	case outboxConflictCountMsg:
		m.outboxConflictCount = msg.count
		return m, nil

	case settingsSyncToggledMsg:
		return m, m.loadVaults()

	case openDownloadPickerMsg:
		// Клик по кликабельной ячейке «Скачать» строки binary-секрета
		m.focus = focusDownloadPicker
		m.downloadPicker = newDownloadPickerPopup(m.currentVaultID(), msg.secretID, msg.filename)
		return m, m.downloadPicker.Init()
	}

	// Сообщения, адресованные таблице (загрузка строк/ошибки), доставляются независимо от
	// текущего фокуса — могут приходить асинхронно, пока focusCommand активен (пользователь
	// печатает в поисковой строке, а фоновый sync/reload уже отдал rowsLoadedMsg).
	switch msg.(type) {
	case rowsErrMsg, rowsLoadedMsg:
		var cmd tea.Cmd
		m.table, cmd = m.table.update(msg)
		return m, cmd
	}

	// Делегируем таблице (raw payload reveal, debounce, etc.) — но только когда она в фокусе.
	if m.focus == focusTable {
		var cmd tea.Cmd
		m.table, cmd = m.table.update(msg)
		return m, cmd
	}

	// Попап скачивания читает каталог асинхронно (readDirMsg от bubbles/filepicker) — эти
	// сообщения не tea.KeyMsg/tea.MouseMsg и без этой ветки не доходили бы до downloadPicker,
	// оставляя список файлов пустым ("Bummer. No Files Found") даже в непустой директории.
	if m.focus == focusDownloadPicker {
		var cmd tea.Cmd
		var done bool
		m.downloadPicker, cmd, done = m.downloadPicker.update(m.ctx, m.container, msg)
		if done {
			m.focus = focusTable
		}
		return m, cmd
	}
	return m, nil
}

func (m dashboardModel) currentVaultID() string {
	v, ok := m.currentVault()
	if !ok {
		return ""
	}
	return v.ID
}

// selectVaultByID возвращает модель с курсором, установленным на vault с указанным ID
// (если он есть в списке). Если ID пуст или не найден — курсор не меняется.
func (m dashboardModel) selectVaultByID(id string) dashboardModel {
	if id == "" {
		return m
	}
	for i, v := range m.vaults {
		if v.ID == id {
			m.vaultCursor = i
			return m
		}
	}
	return m
}

// handleKey — центральная точка обработки клавиатуры dashboard. Шорткаты сверяются через
// normalizeKey/isShortcut, чтобы работать одинаково на любой системной раскладке.
func (m dashboardModel) handleKey(msg tea.KeyMsg) (dashboardModel, tea.Cmd) {
	// Попапы (settings/user/vault-create) перехватывают ввод целиком, пока открыты.
	switch m.focus {
	case focusSettings:
		return m.handleSettingsKey(msg)
	case focusUser:
		return m.handleUserMenuKey(msg)
	case focusVaultCreate:
		return m.handleVaultCreateKey(msg)
	case focusCommand:
		return m.handleCommandKey(msg)
	case focusSyncScope:
		return m.handleSyncScopeKey(msg)
	case focusLogs:
		if msg.Type == tea.KeyEsc || msg.String() == "q" {
			m.focus = focusTable
			return m, nil
		}
		if isShortcut(msg, "d") {
			m.logs = m.logs.clear(m.container.Config.DataDir)
			return m, nil
		}
		m.logs = m.logs.update(msg)
		return m, nil
	case focusDownloadPicker:
		var cmd tea.Cmd
		var done bool
		m.downloadPicker, cmd, done = m.downloadPicker.update(m.ctx, m.container, msg)
		if done {
			m.focus = focusTable
		}
		return m, cmd
	case focusDetail:
		// Больше не используется — detail встроен в форму как view mode.
		m.focus = focusTable
		return m, nil
	}

	// focusTable — основной режим. Переключение type-табов — Tab/Shift+Tab.
	switch msg.Type {
	case tea.KeyTab:
		m.typeCursor = incWrap(m.typeCursor, len(typeTabs(m.l)))
		var cmd tea.Cmd
		m.table, cmd = m.table.setVaultAndType(m.currentVaultID(), m.currentTypeTab().secretType)
		return m, cmd
	case tea.KeyShiftTab:
		m.typeCursor = decWrap(m.typeCursor, len(typeTabs(m.l)))
		var cmd tea.Cmd
		m.table, cmd = m.table.setVaultAndType(m.currentVaultID(), m.currentTypeTab().secretType)
		return m, cmd
	}

	// Прямой выбор vault по цифре 1-9 (физические клавиши одинаковы на любой раскладке).
	if len(msg.Runes) == 1 && msg.Runes[0] >= '1' && msg.Runes[0] <= '9' {
		idx := int(msg.Runes[0] - '1')
		if idx < len(m.vaults) {
			m.vaultCursor = idx
			var cmd tea.Cmd
			m.table, cmd = m.table.setVaultAndType(m.currentVaultID(), m.currentTypeTab().secretType)
			return m, cmd
		}
	}

	switch {
	// Переключение vault-табов — «[» (пред.) / «]» (след.). Ctrl+←/→ не используем: их
	// перехватывает macOS (Mission Control). «[»/«]» нормализуются через раскладку (х/ъ).
	case isShortcut(msg, "["):
		m.vaultCursor = decWrap(m.vaultCursor, len(m.vaults))
		var cmd tea.Cmd
		m.table, cmd = m.table.setVaultAndType(m.currentVaultID(), m.currentTypeTab().secretType)
		return m, cmd
	case isShortcut(msg, "]"):
		m.vaultCursor = incWrap(m.vaultCursor, len(m.vaults))
		var cmd tea.Cmd
		m.table, cmd = m.table.setVaultAndType(m.currentVaultID(), m.currentTypeTab().secretType)
		return m, cmd
	case isShortcut(msg, "/"):
		// "/" открывает нижнюю строку пустой — обычный режим живого поиска. Если пользователь
		// сам напечатает "/" первым символом — строка переключится в командную палитру
		m.focus = focusCommand
		m.commandLine = m.commandLine.activate()
		return m, nil
	case isShortcut(msg, "s"):
		// «Настройки [S]»
		m.focus = focusSettings
		m.settings = newSettingsPopup(m.container)
		rows := make([]settingsVaultRow, 0, len(m.vaults))
		for _, v := range m.vaults {
			rows = append(rows, settingsVaultRow{ID: v.ID, Name: v.Name, SyncEnabled: v.SyncEnabled})
		}
		m.settings = m.settings.SetVaults(m.ctx, rows)
		return m, nil
	case isShortcut(msg, "u"):
		m.focus = focusUser
		return m, nil
	case isShortcut(msg, "g"):
		m.focus = focusLogs
		m.logs = newLogsPopup(m.container.Config.DataDir)
		return m, nil
	case isShortcut(msg, "x"):
		// «Конфликты [x]» — переход к разрешению outbox-конфликтов, обнаруженных фоновым
		// ReplayOutbox (гонка версий с другим устройством, не показывалась пользователю сама).
		return m, m.openFirstOutboxConflict()
	case isShortcut(msg, "l"):
		// Мягкая блокировка: сохраняем PIN, чтобы вернуться по PIN без полного пароля.
		m.container.Session.SoftLock()
		return m, tea.Batch(
			showToast(m.l.T("tui_toast_locked")),
			func() tea.Msg { return switchScreenMsg{screen: screenLock} },
		)
	case isShortcut(msg, "n"):
		if _, ok := m.currentVault(); !ok {
			m.focus = focusVaultCreate
			m.vaultCreate = newVaultCreateModel()
			return m, nil
		}
		vaultID, vaultName, preselect := m.currentVaultID(), m.currentVaultName(), m.currentTypeTab().secretType
		return m, func() tea.Msg {
			return switchScreenMsg{screen: screenForm, vaultID: vaultID, vaultName: vaultName, preselectType: preselect}
		}
	case msg.Type == tea.KeyCtrlN:
		// Ctrl+N создаёт НОВЫЙ vault всегда в отличие от "n", который при непустом списке создаёт секрет в текущем.
		m.focus = focusVaultCreate
		m.vaultCreate = newVaultCreateModel()
		return m, nil
	}

	if len(m.vaults) == 0 {
		// Нет папок — единственное разумное действие — создать (n) или выйти.
		return m, nil
	}

	// [V] — открыть секрет в view mode (то же что Enter — оба открывают форму).
	if isShortcut(msg, "v") {
		id := m.table.currentSecretID()
		if id != "" {
			return m, m.table.openForm()
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.update(msg)
	return m, cmd
}

func (m dashboardModel) currentVaultName() string {
	v, ok := m.currentVault()
	if !ok {
		return ""
	}
	return v.Name
}

// loadVaults — быстрая загрузка из локального кеша (для рефрешей после действий пользователя).
// При ошибке локального кеша падает на серверный List.
func (m dashboardModel) loadVaults() tea.Cmd {
	container := m.container
	ctx := m.ctx
	return func() tea.Msg {
		vaults, err := container.Vault.ListLocal(ctx)
		if err != nil {
			vaults, err = container.Vault.List(ctx)
			if err != nil {
				return vaultsErrMsg{err: err}
			}
		}
		return vaultsLoadedMsg{vaults: vaults}
	}
}

// loadVaultsFromServer — загрузка с сервера (Vault.List кеширует локально + возвращает).
// Используется на первой загрузке dashboard, чтобы обнаружить удалённые папки на новом
// устройстве. Оффлайн (сеть недоступна) — падает на локальный кеш.
func (m dashboardModel) loadVaultsFromServer() tea.Cmd {
	container := m.container
	ctx := m.ctx
	return func() tea.Msg {
		vaults, err := container.Vault.List(ctx)
		if err != nil {
			vaults, lerr := container.Vault.ListLocal(ctx)
			if lerr != nil {
				return vaultsErrMsg{err: err}
			}
			return vaultsLoadedMsg{vaults: vaults}
		}
		return vaultsLoadedMsg{vaults: vaults}
	}
}

func incWrap(i, n int) int {
	if n == 0 {
		return 0
	}
	return (i + 1) % n
}

func decWrap(i, n int) int {
	if n == 0 {
		return 0
	}
	return (i - 1 + n) % n
}

func (m dashboardModel) handleSettingsKey(msg tea.KeyMsg) (dashboardModel, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.focus = focusTable
		return m, nil
	}
	var cmd tea.Cmd
	m.settings, cmd = m.settings.update(msg)
	return m, cmd
}

func (m dashboardModel) handleUserMenuKey(msg tea.KeyMsg) (dashboardModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.focus = focusTable
		return m, nil
	case tea.KeyEnter:
		// Log out: чистим сессию и токены, возвращаемся на экран логина.
		m.container.Session.Lock()
		m.focus = focusTable
		return m, func() tea.Msg { return switchScreenMsg{screen: screenLogin} }
	}
	return m, nil
}

// handleSyncScopeKey обрабатывает попап выбора папок для синхронизации. Esc допускается —
// в этом случае считаем выбор «всё включено по умолчанию» (тот же эффект, что подтверждение
// без изменений), чтобы попап не мог зациклить пользователя без выхода.
func (m dashboardModel) handleSyncScopeKey(msg tea.KeyMsg) (dashboardModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyEnter:
		return m, m.syncScope.confirm(m.ctx, m.container)
	}
	m.syncScope = m.syncScope.update(msg)
	return m, nil
}

func (m dashboardModel) handleVaultCreateKey(msg tea.KeyMsg) (dashboardModel, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.focus = focusTable
		return m, nil
	}
	if msg.Type == tea.KeyEnter {
		name := m.vaultCreate.value()
		if name == "" {
			return m, nil
		}
		m.focus = focusTable
		return m, m.doCreateVault(name)
	}
	var cmd tea.Cmd
	m.vaultCreate, cmd = m.vaultCreate.update(msg)
	return m, cmd
}

func (m dashboardModel) doCreateVault(name string) tea.Cmd {
	container := m.container
	ctx := m.ctx
	return func() tea.Msg {
		id, err := container.Vault.Create(ctx, name)
		if err != nil {
			return vaultsErrMsg{err: err}
		}
		return vaultCreatedMsg{id: id}
	}
}

// handleCommandKey обрабатывает ввод нижней строки. Открывается по "/" сразу в командном
// режиме — с палитрой команд (sync/new/vault/lock/logs/conflicts/quit), Tab автодополняет
// выделенную подсказку, ↑↓ перемещают выделение по палитре, Enter выполняет выделенную/
// введённую команду. Если пользователь стирает "/" и печатает произвольный текст — строка
// становится обычным живым поиском по таблице (isCommand() перестаёт быть true).
func (m dashboardModel) handleCommandKey(msg tea.KeyMsg) (dashboardModel, tea.Cmd) {
	if m.commandLine.isCommand() {
		switch msg.Type {
		case tea.KeyEsc:
			m.focus = focusTable
			m.commandLine = m.commandLine.reset()
			var cmd tea.Cmd
			m.table, cmd = m.table.clearSearch()
			return m, cmd
		case tea.KeyUp:
			m.commandLine = m.commandLine.moveSuggestion(-1)
			return m, nil
		case tea.KeyDown:
			m.commandLine = m.commandLine.moveSuggestion(1)
			return m, nil
		case tea.KeyTab:
			m.commandLine = m.commandLine.acceptSuggestion()
			return m, nil
		case tea.KeyEnter:
			cmdName := m.commandLine.commandToRun()
			m.commandLine = m.commandLine.reset()
			m.focus = focusTable
			return m.runCommand(cmdName)
		}
		var cmd tea.Cmd
		m.commandLine, cmd = m.commandLine.update(msg)
		return m, cmd
	}

	switch msg.Type {
	case tea.KeyEsc:
		m.focus = focusTable
		m.commandLine = m.commandLine.reset()
		var cmd tea.Cmd
		m.table, cmd = m.table.clearSearch()
		return m, cmd
	case tea.KeyEnter:
		// Поиск подтверждён — остаёмся в командной строке, фокус можно вернуть в таблицу
		// для навигации по результатам, не сбрасывая запрос.
		m.focus = focusTable
		return m, nil
	case tea.KeyUp, tea.KeyDown:
		// Стрелки ↑↓ навигируют по строкам таблицы, пока поисковая строка открыта.
		var tCmd tea.Cmd
		m.table, tCmd = m.table.update(msg)
		return m, tCmd
	}

	var cmd tea.Cmd
	m.commandLine, cmd = m.commandLine.update(msg)
	if !m.commandLine.isCommand() {
		var searchCmd tea.Cmd
		m.table, searchCmd = m.table.setSearchQuery(m.commandLine.searchQuery())
		return m, tea.Batch(cmd, searchCmd)
	}
	return m, cmd
}

// runCommand выполняет именованную команду командной строки (sync, new, vault, lock, logs,
// conflicts, quit — плюс алиасы q/log/conflict для совместимости).
func (m dashboardModel) runCommand(raw string) (dashboardModel, tea.Cmd) {
	cmdName := strings.TrimSpace(raw)
	switch cmdName {
	case "sync":
		m.syncing = true
		return m, m.doSync()
	case "new":
		if _, ok := m.currentVault(); !ok {
			m.focus = focusVaultCreate
			m.vaultCreate = newVaultCreateModel()
			return m, nil
		}
		vaultID, vaultName, preselect := m.currentVaultID(), m.currentVaultName(), m.currentTypeTab().secretType
		return m, func() tea.Msg {
			return switchScreenMsg{screen: screenForm, vaultID: vaultID, vaultName: vaultName, preselectType: preselect}
		}
	case "vault":
		// В отличие от ":new" — всегда создаёт vault, даже если текущие уже есть.
		m.focus = focusVaultCreate
		m.vaultCreate = newVaultCreateModel()
		return m, nil
	case "lock":
		m.container.Session.SoftLock()
		return m, func() tea.Msg { return switchScreenMsg{screen: screenLock} }
	case "quit", "q":
		return m, tea.Quit
	case "logs", "log":
		m.focus = focusLogs
		m.logs = newLogsPopup(m.container.Config.DataDir)
		return m, nil
	case "conflicts", "conflict":
		return m, m.openFirstOutboxConflict()
	}
	return m, nil
}

// openFirstOutboxConflict ищет первую нерешённую outbox-запись со статусом conflict и открывает для неё
// общий экран разрешения конфликтов (screenConflict). Если конфликтов нет — показывает тост.
func (m dashboardModel) openFirstOutboxConflict() tea.Cmd {
	container := m.container
	ctx := m.ctx
	return func() tea.Msg {
		entries, err := container.Secret.ListOutboxConflicts(ctx)
		if err != nil {
			return toastMsg{text: fmt.Sprintf("conflicts: %v", err)}
		}
		if len(entries) == 0 {
			return toastMsg{text: container.Localizer.T("tui_toast_no_conflicts")}
		}
		conflict, err := container.Secret.ConflictFromOutbox(ctx, entries[0].ID)
		if err != nil {
			return toastMsg{text: fmt.Sprintf("conflicts: %v", err)}
		}
		if conflict == nil {
			// Гонка разрешилась сама пробуем ещё раз.
			return toastMsg{text: container.Localizer.T("tui_toast_conflict_autoresolved")}
		}
		return switchScreenMsg{screen: screenConflict, conflict: conflict, vaultID: conflict.VaultID}
	}
}

func (m dashboardModel) doSync() tea.Cmd {
	container := m.container
	ctx := m.ctx
	return func() tea.Msg {
		if err := container.Sync.Sync(ctx); err != nil {
			slog.Warn("manual sync failed", "err", err)
			return syncErrMsg{err: err}
		}
		return syncDoneMsg{}
	}
}

// launchBackgroundSync — функциональная версия для использования в Init/Refresh/Tick
// (value receiver). Запускает sync в горутине и возвращает cmd для real-time чтения прогресса.
func (m dashboardModel) launchBackgroundSync() tea.Cmd {
	container := m.container
	ctx := m.ctx

	progressCh := make(chan syncuc.Progress, 8)
	errCh := make(chan error, 1)

	go func() {
		if err := container.Sync.ReplayOutbox(ctx); err != nil {
			slog.Warn("background sync: replay outbox failed", "err", err)
			errCh <- err
			close(progressCh)
			return
		}
		errCh <- container.Sync.SyncWithProgress(ctx, progressCh)
	}()

	return waitForSyncProgress(progressCh, errCh)
}

// waitForSyncProgress — tea.Cmd который читает один элемент из канала прогресса.
// Если канал открыт — возвращает syncProgressMsg и запланирует следующее чтение.
// Если канал закрыт — читает результат из errCh и возвращает backgroundSyncDoneMsg.
func waitForSyncProgress(progressCh <-chan syncuc.Progress, errCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		p, ok := <-progressCh
		if !ok {
			// Канал закрыт — sync завершён, читаем результат.
			err := <-errCh
			if err != nil {
				slog.Warn("background sync: sync failed", "err", err)
			}
			return backgroundSyncDoneMsg{err: err}
		}
		// Есть прогресс — возвращаем его + планируем следующее чтение.
		return syncProgressWithNext{label: p.String(), progressCh: progressCh, errCh: errCh}
	}
}

// syncProgressWithNext — сообщение прогресса + ссылки для продолжения чтения.
type syncProgressWithNext struct {
	label      string
	progressCh <-chan syncuc.Progress
	errCh      <-chan error
}

// renderTopBar рисует верхнюю строку: логотип слева, Настройки/Юзер/Заблокировать
// справа. Фильтры расположены рядом со строкой поиска, а не тут —
// сюда вынесена только кнопка-напоминание шортката.
func (m dashboardModel) renderTopBar(width int) string {
	left := styles.InputLabel.Render(m.l.T("tui_app_name"))

	// Версия клиента (если задана).
	if v := m.container.Config.Version; v != "" {
		left += " " + styles.HelpText.Render("v"+v)
	}

	// Статус синхронизации: показываем справа от логотипа, чтобы пользователь всегда видел,
	// что происходит с фоновым sync.
	syncStatus := m.syncStatusLabel()
	if syncStatus != "" {
		left += "  " + syncStatus
	}

	right := m.renderTopBarButtons()
	if m.syncing {
		right = m.l.T("tui_syncing") + "  " + right
	}
	gap := width - lipglossWidth(left) - lipglossWidth(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// renderTopBarButtons рендерит кнопки топ-бара с BubbleZone и подсветкой активной.
func (m dashboardModel) renderTopBarButtons() string {
	type btn struct {
		label  string
		zoneID string
		active bool
	}
	userLabel := m.l.T("tui_topbar_user")
	if login := m.container.Auth.CurrentLogin(m.ctx); login != "" {
		userLabel = login + " " + userLabel
	}
	buttons := []btn{
		{m.l.T("tui_topbar_settings"), "topbar_settings", m.focus == focusSettings},
		{m.l.T("tui_topbar_logs"), "topbar_logs", m.focus == focusLogs},
	}
	if m.outboxConflictCount > 0 {
		buttons = append(buttons, btn{
			label:  fmt.Sprintf("⚠ %s (%d)", m.l.T("tui_topbar_conflicts"), m.outboxConflictCount),
			zoneID: "topbar_conflicts",
		})
	}
	buttons = append(buttons,
		btn{userLabel, "topbar_user", m.focus == focusUser},
		btn{m.l.T("tui_topbar_lock"), "topbar_lock", false},
	)
	var parts []string
	for _, b := range buttons {
		var rendered string
		if b.active {
			rendered = styles.TabActive.Render(b.label)
		} else {
			rendered = styles.HelpText.Render(b.label)
		}
		parts = append(parts, zone.Mark(b.zoneID, rendered))
	}
	return strings.Join(parts, "  ")
}

// syncStatusLabel формирует краткую метку состояния sync для топ-бара.
func (m dashboardModel) syncStatusLabel() string {
	if m.backgroundSyncing {
		if m.syncProgressLabel != "" {
			return styles.HelpText.Render(m.syncProgressLabel)
		}
		return styles.HelpText.Render("↻ " + m.l.T("tui_sync_in_progress"))
	}
	if m.lastBackgroundSyncErr != nil {
		// Показываем конкретную ошибку (не просто "offline") — чтобы было ясно, протухли ли
		// токены (нужен re-login) или сервер реально недоступен.
		errText := m.lastBackgroundSyncErr.Error()
		if len(errText) > 40 {
			errText = errText[:40] + "…"
		}
		return styles.ErrorText.Render("✗ " + errText)
	}
	if !m.lastSyncOK.IsZero() {
		ago := time.Since(m.lastSyncOK).Truncate(time.Second)
		return styles.SuccessText.Render("✓ " + m.l.T("tui_sync_ago") + " " + ago.String())
	}
	return styles.HelpText.Render("— " + m.l.T("tui_sync_never"))
}

// isAuthError сообщает, что ошибка вызвана невалидными/протухшими токенами (Unauthenticated).
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "invalid login or credential") ||
		strings.Contains(err.Error(), "token")
}

// renderVaultTabs рисует первый ряд вкладок (папки). Активная — выделена стилем TabActive.
// Перед именем — индикатор синхронизации: ● если синхронизация включена, ○ если отключена.
func (m dashboardModel) renderVaultTabs() string {
	if len(m.vaults) == 0 {
		return styles.HelpText.Render(m.l.T("tui_no_vaults"))
	}
	var parts []string
	for i, v := range m.vaults {
		mark := "○ " // синхронизация выключена
		if v.SyncEnabled {
			mark = "● " // синхронизируется
		}
		label := mark + v.Name
		var rendered string
		if i == m.vaultCursor {
			rendered = styles.TabActive.Render(label)
		} else {
			rendered = styles.TabInactive.Render(label)
		}
		parts = append(parts, zone.Mark(fmt.Sprintf("vault_%d", i), rendered))
	}
	return strings.Join(parts, "")
}

// renderTypeTabs рисует второй ряд вкладок (тип секрета: Все/Пароли/Карточки/Коды/Файлы/Заметки).
// Подсказки о создании вынесены в отдельную нижнюю строку помощи, чтобы ряд табов всегда
// занимал ровно одну строку и не переносился на узких терминалах.
func (m dashboardModel) renderTypeTabs() string {
	tabs := typeTabs(m.l)
	var parts []string
	for i, t := range tabs {
		label := t.label
		// Если вкладка TOTP активна — добавляем общий обратный отсчёт [Xs].
		if i == m.typeCursor && t.secretType == domain.SecretTypeTOTP {
			remaining := 30 - (m.table.totpNow.Second() % 30)
			label = fmt.Sprintf("%s [%ds]", t.label, remaining)
		}
		var rendered string
		if i == m.typeCursor {
			rendered = styles.TabActive.Render(label)
		} else {
			rendered = styles.TabInactive.Render(label)
		}
		parts = append(parts, zone.Mark(fmt.Sprintf("type_%d", i), rendered))
	}
	return strings.Join(parts, "")
}

// lipglossWidth — визуальная ширина строки без учёта ANSI-стилей.
func lipglossWidth(s string) int {
	return len([]rune(ansiEscapeRe.ReplaceAllString(s, "")))
}

// handleTabClick обрабатывает клик мыши по vault-табам, type-табам и кнопкам топ-бара (BubbleZone).
func (m dashboardModel) handleTabClick(msg tea.MouseMsg) (dashboardModel, tea.Cmd) {
	// Клик по кнопкам топ-бара.
	if zone.Get("topbar_settings").InBounds(msg) {
		m.focus = focusSettings
		m.settings = newSettingsPopup(m.container)
		rows := make([]settingsVaultRow, 0, len(m.vaults))
		for _, v := range m.vaults {
			rows = append(rows, settingsVaultRow{ID: v.ID, Name: v.Name, SyncEnabled: v.SyncEnabled})
		}
		m.settings = m.settings.SetVaults(m.ctx, rows)
		return m, nil
	}
	if zone.Get("topbar_logs").InBounds(msg) {
		m.focus = focusLogs
		m.logs = newLogsPopup(m.container.Config.DataDir)
		return m, nil
	}
	if zone.Get("topbar_conflicts").InBounds(msg) {
		return m, m.openFirstOutboxConflict()
	}
	if zone.Get("topbar_user").InBounds(msg) {
		m.focus = focusUser
		return m, nil
	}
	if zone.Get("topbar_lock").InBounds(msg) {
		m.container.Session.SoftLock()
		return m, func() tea.Msg { return switchScreenMsg{screen: screenLock} }
	}

	// Клик по vault-табу.
	for i := range m.vaults {
		zoneID := fmt.Sprintf("vault_%d", i)
		if zone.Get(zoneID).InBounds(msg) {
			m.vaultCursor = i
			var cmd tea.Cmd
			m.table, cmd = m.table.setVaultAndType(m.currentVaultID(), m.currentTypeTab().secretType)
			return m, cmd
		}
	}
	// Клик по type-табу.
	tabs := typeTabs(m.l)
	for i := range tabs {
		zoneID := fmt.Sprintf("type_%d", i)
		if zone.Get(zoneID).InBounds(msg) {
			m.typeCursor = i
			var cmd tea.Cmd
			m.table, cmd = m.table.setVaultAndType(m.currentVaultID(), m.currentTypeTab().secretType)
			return m, cmd
		}
	}
	return m, nil
}
