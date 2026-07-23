package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
)

// screenID определяет текущий «экран» TUI. Dashboard — единственный persistent-экран после
// разблокировки: вкладки папок/типов секретов переключаются внутри него без смены screenID.
// Остальные screenID — модальные/полноэкранные шаги вокруг него (auth, форма, конфликт).
type screenID int

const (
	screenLogin           screenID = iota // Логин / регистрация
	screenLock                            // Разблокировка (master password)
	screenSetupEncryption                 // Первичная настройка шифрования
	screenDashboard                       // Persistent dashboard (топ-бар + вкладки + таблица)
	screenForm                            // Форма создания/редактирования секрета
	screenConflict                        // Экран разрешения конфликта
	screenRecoveryCodes                   // Показ recovery codes после настройки шифрования
)

// App — корневая модель TUI-приложения (state machine).
type App struct {
	ctx       context.Context
	container *app.Container

	screen screenID
	width  int
	height int

	// autolockTimeout — текущий таймаут автоблокировки (0 = никогда). Инициализируется из
	// config.AutolockMinutes, меняется в рантайме через autolockChangedMsg из Settings.
	autolockTimeout time.Duration

	login           loginModel
	lock            lockModel
	setupEncryption setupEncryptionModel
	dashboard       dashboardModel
	form            secretFormModel
	conflict        conflictModel
	recoveryCodes   recoveryCodesModel

	toast toastModel

	lastActivity time.Time
}

// New создаёт корневую TUI-модель. startScreen определяется вызывающим кодом
// на основании состояния конфигурации и сессии.
func New(ctx context.Context, container *app.Container, startScreen screenID) App {
	a := App{
		ctx:             ctx,
		container:       container,
		screen:          startScreen,
		lastActivity:    time.Now(),
		autolockTimeout: autolockTimeoutFromConfig(container.Config.AutolockMinutes),
	}
	a.login = newLoginModel(container)
	a.lock = newLockModel(container)
	a.setupEncryption = newSetupEncryptionModel(container)
	a.dashboard = newDashboardModel(ctx, container)
	a.form = newSecretFormModel(ctx, container)
	a.conflict = newConflictModel(ctx, container)
	a.toast = newToastModel()
	return a
}

// Init инициализирует модель App, запускает таймер автоблокировки и вызывает Init активного экрана.
func (a App) Init() tea.Cmd {
	cmds := []tea.Cmd{a.autolockTick()}
	switch a.screen {
	case screenLogin:
		cmds = append(cmds, a.login.Init())
	case screenLock:
		cmds = append(cmds, a.lock.Init())
	case screenDashboard:
		cmds = append(cmds, a.dashboard.Init())
	}
	return tea.Batch(cmds...)
}

// Update обрабатывает входящие сообщения и направляет их в модель активного экрана.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		a.lastActivity = time.Now()
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// При ресайзе (например пользователь изменил размер панели терминала в IDE) форсим
		// полную очистку экрана перед следующим кадром — иначе может остаться "хвост" от
		// предыдущего кадра, отрендеренного под старую ширину (особенно заметно на таблице
		// с адаптивными колонками, у которой меняется реальная длина строк между кадрами).
		cmds = append(cmds, tea.ClearScreen)

	case switchScreenMsg:
		a.screen = msg.screen
		var ctxCmd tea.Cmd
		a, ctxCmd = a.applyScreenContext(msg)
		cmds = append(cmds, ctxCmd)

	case recoveryCodesGeneratedMsg:
		a.recoveryCodes = newRecoveryCodesModel(a.container, msg.codes)
		a.screen = screenRecoveryCodes

	case autolockMsg:
		// Проверка идёт здесь (по актуальным a.lastActivity/a.autolockTimeout), а сам тик
		// всегда перепланируется — иначе автолок переставал бы тикать после первого тика.
		if a.autolockTimeout > 0 && time.Since(a.lastActivity) >= a.autolockTimeout &&
			a.screen != screenLogin && a.screen != screenLock && a.screen != screenSetupEncryption {
			// Мягкая блокировка: MasterKey стирается, но PIN-материал сохраняется, чтобы
			// пользователь мог разблокироваться PIN'ом без полного master-пароля.
			a.container.Session.SoftLock()
			a.screen = screenLock
			a.lock = newLockModel(a.container)
			cmds = append(cmds, a.lock.Init())
		}
		cmds = append(cmds, a.autolockTick())

	case autolockChangedMsg:
		a.autolockTimeout = msg.timeout
	}

	var toastCmd tea.Cmd
	a.toast, toastCmd = a.toast.update(msg)
	if toastCmd != nil {
		cmds = append(cmds, toastCmd)
	}

	// Фоновый sync (tick/progress/done) — сквозной процесс, не привязанный к тому, какой
	// экран сейчас активен (пользователь может открыть форму/lock/settings пока идёт sync).
	// Если не обрабатывать эти сообщения здесь, они дропаются на экранах != screenDashboard,
	// а значит следующий tea.Tick никогда не будет перепланирован — фоновый sync навсегда
	// останавливается до перезапуска клиента.
	switch msg.(type) {
	case backgroundSyncTickMsg, syncProgressWithNext, backgroundSyncDoneMsg, outboxConflictCountMsg:
		var dashCmd tea.Cmd
		a.dashboard, dashCmd = a.dashboard.update(msg)
		return a, tea.Batch(append(cmds, dashCmd)...)
	}

	var cmd tea.Cmd
	switch a.screen {
	case screenLogin:
		a.login, cmd = a.login.update(msg)
	case screenLock:
		a.lock, cmd = a.lock.update(msg)
	case screenSetupEncryption:
		a.setupEncryption, cmd = a.setupEncryption.update(msg)
	case screenDashboard:
		a.dashboard, cmd = a.dashboard.update(msg)
	case screenForm:
		a.form, cmd = a.form.update(msg)
	case screenConflict:
		a.conflict, cmd = a.conflict.update(msg)
	case screenRecoveryCodes:
		a.recoveryCodes, cmd = a.recoveryCodes.update(msg)
	}
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

// View renders the currently active screen, plus any visible toast notification.
func (a App) View() string {
	var body string
	switch a.screen {
	case screenLogin:
		body = a.login.view(a.width, a.height)
	case screenLock:
		body = a.lock.view(a.width, a.height)
	case screenSetupEncryption:
		body = a.setupEncryption.view(a.width, a.height)
	case screenDashboard:
		body = a.dashboard.view(a.width, a.height)
	case screenForm:
		body = a.form.view(a.width, a.height)
	case screenConflict:
		body = a.conflict.view(a.width, a.height)
	case screenRecoveryCodes:
		body = a.recoveryCodes.view(a.width, a.height)
	}

	if a.toast.visible {
		body += "\n" + a.toast.view()
	}
	return zone.Scan(body)
}

// applyScreenContext переносит контекст навигации (msg) в дочернюю модель вновь
// активированного экрана и вызывает её Init.
func (a App) applyScreenContext(msg switchScreenMsg) (App, tea.Cmd) {
	switch msg.screen {
	case screenLogin:
		return a, a.login.Init()
	case screenLock:
		// Пересоздаём lock-модель, чтобы сбросить прежний режим (setPIN/ошибки) и заново
		// определить PIN/password-режим по текущему состоянию сессии.
		a.lock = newLockModel(a.container)
		return a, a.lock.Init()
	case screenSetupEncryption:
		return a, a.setupEncryption.Init()
	case screenDashboard:
		// Dashboard хранит своё состояние (текущий vault/type/cursor) само по себе —
		// не пересоздаём модель, просто просим её обновить данные при возврате. На первом
		// входе (после login/unlock) Refresh грузит папки с сервера, чтобы обнаружить
		// удалённые папки на новом устройстве; далее — быстрый локальный рефреш.
		// vaultID из msg позволяет вернуть курсор на vault, в котором пользователь работал.
		a.dashboard = a.dashboard.selectVaultByID(msg.vaultID)
		return a, a.dashboard.Refresh()
	case screenForm:
		if msg.editSecret {
			a.form = a.form.SetEditDataWithMode(msg.vaultID, msg.vaultName, msg.secretID, msg.secretVer, msg.editType, msg.editData, msg.openInEditMode)
		} else {
			a.form = a.form.SetCreate(msg.vaultID, msg.vaultName, msg.preselectType)
		}
		return a, a.form.Init()
	case screenConflict:
		a.conflict = a.conflict.SetConflict(msg.conflict, msg.vaultID, msg.vaultName)
		return a, a.conflict.Init()
	}
	return a, nil
}

// autolockTick планирует периодическую проверку бездействия (раз в минуту). Сам тик всегда
// шлёт autolockMsg — решение о блокировке принимается в обработчике по актуальному состоянию.
func (a App) autolockTick() tea.Cmd {
	return tea.Tick(time.Minute, func(time.Time) tea.Msg {
		return autolockMsg{}
	})
}

// autolockTimeoutFromConfig переводит минуты из конфига в Duration. <=0 → «никогда» (0).
func autolockTimeoutFromConfig(minutes int) time.Duration {
	if minutes <= 0 {
		return 0
	}
	return time.Duration(minutes) * time.Minute
}

// StartScreen определяет стартовый экран на основании текущего состояния сессии.
// Onboarding-визарда нет: конфиг (server addr/data dir/lang) настраивается так же,
// как у CLI — через config.json/env/флаги, до создания Container.
func StartScreen(container *app.Container) screenID {
	if container.Session.Unlocked() {
		return screenDashboard
	}
	if container.Auth.EncryptionConfigured() {
		return screenLock
	}
	return screenLogin
}
