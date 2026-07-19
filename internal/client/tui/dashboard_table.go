package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

const revealDebounce = 250 * time.Millisecond

type debounceTickMsg struct{ secretID string }

type dashboardTableModel struct {
	ctx       context.Context
	container *app.Container

	vaultID    string
	secretType domain.SecretType // 0 = все типы

	allRows    []secret.SummaryRow
	rows       []secret.SummaryRow
	cursor     int
	err        error
	loading    bool
	searchTerm string

	// Раскрытие по фокусу (payload текущей строки).
	revealedID     string
	revealedSecret string

	// Доп. раскрытие карты по [R]: CVV/PIN/полный номер (для строки revealedID).
	cardReveal bool
	cardCVV    string
	cardPIN    string
	cardPAN    string

	revealErr error

	// TOTP: время последнего тика (для обратного отсчёта) и режим показа.
	totpNow      time.Time
	totpAllCodes map[string]string // secretID → код (режим «all»)

	// Визуальная обратная связь «скопировано» — показывает ✓ в ячейке на 1.5с.
	copiedRow int
	copiedCol int
	copiedVis bool // true = сейчас показываем ✓ в (copiedRow, copiedCol)
}

func newDashboardTableModel(ctx context.Context, container *app.Container) dashboardTableModel {
	return dashboardTableModel{ctx: ctx, container: container, loading: true, totpNow: time.Now()}
}

// setVaultAndType переключает контекст таблицы и перезагружает данные, возвращая обновлённую
// модель и команду. Вызывающая сторона должна переприсвоить модель: m = m.setVaultAndType(...).
func (m dashboardTableModel) setVaultAndType(vaultID string, t domain.SecretType) (dashboardTableModel, tea.Cmd) {
	changed := m.vaultID != vaultID || m.secretType != t
	m.vaultID = vaultID
	m.secretType = t

	// Пустой vault — сбрасываем всё безусловно.
	if vaultID == "" {
		m.cursor = 0
		m = m.clearReveal()
		m.searchTerm = ""
		m.totpAllCodes = nil
		m.loading = false
		m.rows = nil
		m.allRows = nil
		return m, nil
	}

	if !changed {
		// Vault/type не изменились — не сбрасываем поиск, курсор и не перезагружаем
		// данные (Refresh вызывает reload() отдельно при необходимости).
		return m, nil
	}
	m.cursor = 0
	m = m.clearReveal()
	m.searchTerm = ""
	m.totpAllCodes = nil
	m.loading = true
	if t == domain.SecretTypeTOTP {
		return m, tea.Batch(m.reload(), m.totpTick())
	}
	return m, m.reload()
}

// clearReveal затирает всё раскрытое чувствительное значение (payload + карта).
func (m dashboardTableModel) clearReveal() dashboardTableModel {
	m.revealedID = ""
	m.revealedSecret = ""
	m.cardReveal = false
	m.cardCVV, m.cardPIN, m.cardPAN = "", "", ""
	return m
}

func (m dashboardTableModel) reload() tea.Cmd {
	if m.vaultID == "" {
		return nil
	}
	container := m.container
	ctx := m.ctx
	vaultID := m.vaultID
	secretType := m.secretType
	return func() tea.Msg {
		rows, err := container.Secret.ListRowsByType(ctx, vaultID, secretType)
		if err != nil {
			return rowsErrMsg{err: err}
		}
		return rowsLoadedMsg{rows: rows}
	}
}

// setSearchQuery применяет живой фильтр. allRows уже расшифрованы и лежат в памяти (загружены
// через reload()/ListRowsByType, включая Tier 2b-поля из enrichSummaryRow) — поэтому фильтрация
// чисто локальная, без похода в usecase/сеть.
func (m dashboardTableModel) setSearchQuery(q string) (dashboardTableModel, tea.Cmd) {
	m.searchTerm = q
	m = m.applyLocalFilter()
	return m, nil
}

func (m dashboardTableModel) clearSearch() (dashboardTableModel, tea.Cmd) {
	m.searchTerm = ""
	m = m.applyLocalFilter()
	return m, nil
}

// applyLocalFilter фильтрует уже загруженные allRows по searchTerm — без сети и повторной
// расшифровки. Матчит по Title/Subtitle/Tags/URI/Bank/Cardholder (все поля, уже присутствующие
// в SummaryRow после enrichSummaryRow).
func (m dashboardTableModel) applyLocalFilter() dashboardTableModel {
	if m.searchTerm == "" {
		m.rows = m.allRows
		if m.cursor >= len(m.rows) {
			m.cursor = max(0, len(m.rows)-1)
		}
		return m
	}
	q := strings.ToLower(m.searchTerm)
	filtered := make([]secret.SummaryRow, 0, len(m.allRows))
	for _, r := range m.allRows {
		if matchesRow(r, q) {
			filtered = append(filtered, r)
		}
	}
	m.rows = filtered
	m.cursor = 0
	return m
}

// matchesRow проверяет, содержит ли строка (любое из отображаемых текстовых полей) подстроку q.
func matchesRow(r secret.SummaryRow, qLower string) bool {
	fields := []string{r.Title, r.Subtitle, r.URI, r.Bank, r.Cardholder, r.Brand}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), qLower) {
			return true
		}
	}
	for _, tag := range r.Tags {
		if strings.Contains(strings.ToLower(tag), qLower) {
			return true
		}
	}
	return false
}

func (m dashboardTableModel) update(msg tea.Msg) (dashboardTableModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionRelease {
			return m, nil
		}
		// Проверяем, попал ли клик в одну из ячеек таблицы (zone "cell_<row>_<col>").
		return m.handleCellClick(msg)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
				m = m.clearReveal() // курсор ушёл со строки — затираем раскрытый payload
				return m, m.scheduleReveal()
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor < len(m.rows)-1 {
				m.cursor++
				m = m.clearReveal()
				return m, m.scheduleReveal()
			}
			return m, nil
		case tea.KeyEnter:
			return m, m.openForm()
		}
		if isShortcut(msg, "c") {
			cmd := m.copyCurrent()
			if cmd != nil {
				// Показываем ✓ в payload-колонке текущей строки. У bank_card и binary теперь
				// последняя колонка — не payload, а отдельное действие («Показать»/«Скачать»),
				// поэтому индекс payload-колонки вычисляется отдельно.
				cols := columnsForType(m.secretType, m.container.Localizer)
				m.copiedRow, m.copiedCol, m.copiedVis = m.cursor, payloadColumnIndex(m.secretType, len(cols)), true
				return m, cmd
			}
			return m, nil
		}
		if isShortcut(msg, "r") {
			// [R] — раскрыть CVV/PIN/полный номер карты (по требованию, отдельно от фокуса).
			if m.secretTypeAt(m.cursor) == domain.SecretTypeBankCard {
				return m, m.revealCard(m.currentSecretID())
			}
		}
		if isShortcut(msg, "d") {
			// [D] — для binary-строки: открыть флоу скачивания (выбор папки назначения),
			// тот же результат, что клик по кликабельной ячейке «Скачать».
			if m.secretTypeAt(m.cursor) == domain.SecretTypeBinary {
				return m, m.openDownloadPicker(m.cursor)
			}
		}
		if isShortcut(msg, "e") {
			// [E] — открыть текущую строку сразу в edit mode, минуя view mode (то же, что
			// Enter → e из карточки, но за один шорткат прямо из таблицы).
			return m, m.openFormEdit()
		}

	case rowsLoadedMsg:
		m.loading = false
		m.allRows = msg.rows
		m.err = nil
		m = m.applyLocalFilter()
		if len(m.rows) > 0 {
			return m, m.scheduleReveal()
		}

	case rowsErrMsg:
		m.loading = false
		m.err = msg.err

	case debounceTickMsg:
		if msg.secretID == m.currentSecretID() {
			return m, m.revealPayload(msg.secretID)
		}

	case payloadRevealedMsg:
		if msg.secretID == m.currentSecretID() {
			m.revealedID = msg.secretID
			m.revealedSecret = msg.password
			m.revealErr = nil
		}

	case cardRevealedMsg:
		if msg.secretID == m.currentSecretID() {
			m.cardReveal = true
			m.cardCVV, m.cardPIN, m.cardPAN = msg.cvv, msg.pin, msg.pan
		}

	case payloadRevealErrMsg:
		m.revealErr = msg.err

	case copiedCellExpiredMsg:
		m.copiedVis = false

	case totpAllCodesMsg:
		m.totpAllCodes = msg.codes

	case totpTickMsg:
		if m.secretType == domain.SecretTypeTOTP {
			m.totpNow = time.Now()
			cmds := []tea.Cmd{m.totpTick()}
			if m.totpRevealAll() {
				cmds = append(cmds, m.loadAllTOTPCodes())
			} else if m.currentSecretID() != "" {
				cmds = append(cmds, m.revealPayload(m.currentSecretID()))
			}
			return m, tea.Batch(cmds...)
		}
	}
	return m, nil
}

// totpRevealAll сообщает, показывать ли все TOTP-коды сразу (config totp_reveal_mode=all).
func (m dashboardTableModel) totpRevealAll() bool {
	return m.container.Config.TOTPRevealMode == "all"
}

func (m dashboardTableModel) totpTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return totpTickMsg{t: t} })
}

// copyCurrent копирует раскрытое значение текущей строки в буфер (пароль/код/номер).
// copyCurrent копирует раскрытое значение текущей строки в буфер (пароль/код/номер).
// Возвращает non-nil cmd если копирование произошло (для вызывающего кода — сигнал показать ✓).
func (m dashboardTableModel) copyCurrent() tea.Cmd {
	val := m.revealedSecret
	if m.cardReveal && m.cardPAN != "" {
		val = m.cardPAN
	}
	if val == "" {
		return nil
	}
	_ = clipboard.WriteAll(val)
	return m.copiedCellTimer() // сигнал + запуск таймера сброса ✓
}

func (m dashboardTableModel) secretTypeAt(idx int) domain.SecretType {
	if idx < 0 || idx >= len(m.rows) {
		return 0
	}
	return domain.SecretType(m.rows[idx].Type)
}

func (m dashboardTableModel) currentSecretID() string {
	if len(m.rows) == 0 || m.cursor >= len(m.rows) {
		return ""
	}
	return m.rows[m.cursor].ID
}

func (m dashboardTableModel) scheduleReveal() tea.Cmd {
	id := m.currentSecretID()
	if id == "" {
		return nil
	}
	// В режиме TOTP-all раскрытие по фокусу не нужно — коды грузятся все сразу.
	if m.secretType == domain.SecretTypeTOTP && m.totpRevealAll() {
		return m.loadAllTOTPCodes()
	}
	return tea.Tick(revealDebounce, func(time.Time) tea.Msg {
		return debounceTickMsg{secretID: id}
	})
}

// revealPayload расшифровывает основное чувствительное поле секрета под тип строки.
func (m dashboardTableModel) revealPayload(secretID string) tea.Cmd {
	container := m.container
	ctx := m.ctx
	vaultID := m.vaultID
	rowType := m.secretTypeOf(secretID)

	return func() tea.Msg {
		var value string
		var err error
		switch rowType {
		case domain.SecretTypeLoginPassword:
			d, e := container.Secret.GetDetail(ctx, vaultID, secretID)
			value, err = d.Payload.Password, e
		case domain.SecretTypeBankCard:
			d, e := container.Secret.GetBankCardDetail(ctx, vaultID, secretID)
			value, err = d.Payload.PAN, e
		case domain.SecretTypeTOTP:
			d, e := container.Secret.GetTOTPDetail(ctx, vaultID, secretID)
			if e == nil {
				value, err = secret.GenerateTOTPCode(d.Payload)
			} else {
				err = e
			}
		case domain.SecretTypeText:
			d, e := container.Secret.GetTextDetail(ctx, vaultID, secretID)
			value, err = firstLine(d.Payload.Body), e
		case domain.SecretTypeBinary:
			value, err = "", nil
		default:
			return payloadRevealErrMsg{err: fmt.Errorf("unknown secret type")}
		}
		if err != nil {
			return payloadRevealErrMsg{err: err}
		}
		return payloadRevealedMsg{secretID: secretID, password: value}
	}
}

// openDownloadPicker возвращает команду, открывающую попап выбора папки для скачивания
// binary-секрета строки idx (Subtitle строки = оригинальное имя файла).
func (m dashboardTableModel) openDownloadPicker(idx int) tea.Cmd {
	if idx < 0 || idx >= len(m.rows) {
		return nil
	}
	row := m.rows[idx]
	return func() tea.Msg {
		return openDownloadPickerMsg{secretID: row.ID, filename: row.Subtitle}
	}
}

// revealCard раскрывает CVV/PIN/полный номер карты по требованию ([R]).
func (m dashboardTableModel) revealCard(secretID string) tea.Cmd {
	if secretID == "" {
		return nil
	}
	container := m.container
	ctx := m.ctx
	vaultID := m.vaultID
	return func() tea.Msg {
		d, err := container.Secret.GetBankCardDetail(ctx, vaultID, secretID)
		if err != nil {
			return payloadRevealErrMsg{err: err}
		}
		return cardRevealedMsg{secretID: secretID, cvv: d.Payload.CVV, pin: d.Payload.PIN, pan: d.Payload.PAN}
	}
}

// loadAllTOTPCodes генерирует коды для всех TOTP-строк текущей папки (режим reveal=all).
func (m dashboardTableModel) loadAllTOTPCodes() tea.Cmd {
	container := m.container
	ctx := m.ctx
	vaultID := m.vaultID
	rows := m.rows
	return func() tea.Msg {
		codes := make(map[string]string, len(rows))
		for _, r := range rows {
			d, err := container.Secret.GetTOTPDetail(ctx, vaultID, r.ID)
			if err != nil {
				continue
			}
			if code, e := secret.GenerateTOTPCode(d.Payload); e == nil {
				codes[r.ID] = code
			}
		}
		return totpAllCodesMsg{codes: codes}
	}
}

func (m dashboardTableModel) secretTypeOf(secretID string) domain.SecretType {
	for _, r := range m.rows {
		if r.ID == secretID {
			return domain.SecretType(r.Type)
		}
	}
	return m.secretType
}

// openForm открывает форму просмотра/редактирования текущей строки (все типы) в обычном
// read-only view mode (Enter/[v]). Данные секрета подгружаются через usecase-детали и
// передаются в форму через switchScreenMsg.
func (m dashboardTableModel) openForm() tea.Cmd {
	return m.openFormWithMode(false)
}

// openFormEdit — то же, что openForm, но сразу открывает secret в edit mode ([e]), минуя
// промежуточный view mode — самый частый случай, когда пользователь уже знает, что хочет
// поменять поле, и Enter → e экономить один шаг.
func (m dashboardTableModel) openFormEdit() tea.Cmd {
	return m.openFormWithMode(true)
}

func (m dashboardTableModel) openFormWithMode(startInEdit bool) tea.Cmd {
	if len(m.rows) == 0 || m.cursor >= len(m.rows) {
		return nil
	}
	row := m.rows[m.cursor]
	container := m.container
	ctx := m.ctx
	vaultID := m.vaultID
	rowType := domain.SecretType(row.Type)

	return func() tea.Msg {
		ed, err := loadEditData(ctx, container, vaultID, row.ID, rowType, row.Version)
		if err != nil {
			return rowsErrMsg{err: err}
		}
		return switchScreenMsg{
			screen:         screenForm,
			vaultID:        vaultID,
			editSecret:     true,
			secretID:       row.ID,
			secretVer:      row.Version,
			editType:       rowType,
			editData:       ed,
			openInEditMode: startInEdit,
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func (m dashboardTableModel) view(width, maxRows int) string {
	var b strings.Builder
	l := m.container.Localizer

	if m.loading {
		b.WriteString(styles.Subtitle.Render(l.T("tui_loading")))
		return b.String()
	}
	if m.err != nil {
		b.WriteString(styles.ErrorText.Render(fmt.Sprintf(l.T("tui_error_prefix"), m.err)))
		b.WriteString("\n\n")
	}

	if len(m.rows) == 0 {
		b.WriteString(styles.Subtitle.Render(l.T("tui_no_secrets")))
		return b.String()
	}

	cols := columnsForType(m.secretType, l)
	// Доступная ширина для колонок: минус курсор ("  "/"▸ " = 2 символа) и минус по одному
	// разделительному пробелу на каждую колонку (padTo уже добавляет минимум 1 пробел справа).
	colWidths := computeColumnWidths(cols, width-2)

	// Заголовок таблицы.
	var head strings.Builder
	head.WriteString("  ")
	for i, c := range cols {
		head.WriteString(padTo(c.title, colWidths[i]))
	}
	if len(m.rows) > maxRows {
		_, _ = fmt.Fprintf(&head, " [%d/%d]", m.cursor+1, len(m.rows))
	}
	b.WriteString(styles.Subtitle.Render(head.String()))
	b.WriteString("\n")
	lineWidth := 2
	for _, w := range colWidths {
		lineWidth += w
	}
	b.WriteString(strings.Repeat("─", min(lineWidth, width)))
	b.WriteString("\n")

	// Windowed scrolling: показываем только maxRows строк вокруг курсора.
	startRow, endRow := scrollWindow(m.cursor, len(m.rows), maxRows)

	for i := startRow; i < endRow; i++ {
		row := m.rows[i]
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}
		// rowType для rowCells: на вкладке All (m.secretType==0) передаём 0, чтобы попасть в
		// обобщённый default case (Title/Type/Info/Value). На конкретной вкладке — её тип.
		rowType := m.secretType

		revealedActive := i == m.cursor
		revealed := ""
		if revealedActive && m.revealedID == row.ID {
			revealed = m.revealedSecret
		}
		// В режиме TOTP-all код показывается для всех строк, не только под фокусом.
		if m.secretType == domain.SecretTypeTOTP && m.totpRevealAll() {
			revealed = m.totpAllCodes[row.ID]
			revealedActive = revealed != ""
		}

		cells := rowCells(rowType, row, revealed, revealedActive, l)
		var line strings.Builder
		line.WriteString(cursor)
		for ci := range cols {
			w := colWidths[ci]
			val := ""
			if ci < len(cells) {
				val = cells[ci]
			}
			// Визуальная обратная связь: показываем ✓ в ячейке зелёным.
			isCopied := m.copiedVis && i == m.copiedRow && ci == m.copiedCol
			if isCopied {
				val = "⎘ copied"
			}
			cellContent := padTo(truncate(val, w-1), w)
			if isCopied {
				cellContent = styles.SuccessText.Render(cellContent)
			}
			// Каждая ячейка — кликабельная зона (bubblezone): клик копирует значение.
			zoneID := fmt.Sprintf("cell_%d_%d", i, ci)
			line.WriteString(zone.Mark(zoneID, cellContent))
		}
		// TOTP: обратный отсчёт встроен в ячейку кода (не отдельная колонка).

		if i == m.cursor {
			b.WriteString(styles.InputLabel.Render(line.String()))
		} else {
			b.WriteString(line.String())
		}
		b.WriteString("\n")
	}

	// Раскрытые по [R] поля карты (под таблицей, для текущей строки).
	if m.cardReveal {
		b.WriteString("\n")
		b.WriteString(styles.InputLabel.Render(fmt.Sprintf(l.T("tui_card_reveal_label"), m.cardPAN, m.cardCVV, m.cardPIN)))
		b.WriteString("\n")
	}

	if m.revealErr != nil {
		b.WriteString("\n")
		b.WriteString(styles.ErrorText.Render(fmt.Sprintf(l.T("tui_reveal_error"), m.revealErr)))
	}
	return b.String()
}

// scrollWindow вычисляет диапазон [start, end) видимых строк при скролле по курсору.
func scrollWindow(cursor, total, maxVisible int) (start, end int) {
	if total <= maxVisible {
		return 0, total
	}
	// Держим курсор примерно в центре видимого окна.
	half := maxVisible / 2
	start = cursor - half
	if start < 0 {
		start = 0
	}
	end = start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
	}
	return start, end
}

func padTo(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return s + " "
	}
	return s + strings.Repeat(" ", width-len(r))
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len([]rune(s)) <= maxLen {
		return s
	}
	r := []rune(s)
	if maxLen <= 1 {
		return string(r[:maxLen])
	}
	return string(r[:maxLen-1]) + "…"
}

// handleCellClick обрабатывает клик мышью по ячейке таблицы (bubblezone). Если клик попал
// в зону "cell_<row>_<col>" — копирует значение ячейки в буфер обмена. Для маскированных
// полей (пароль/CVV/PAN) расшифровывает payload и копирует реальное значение.
func (m dashboardTableModel) handleCellClick(msg tea.MouseMsg) (dashboardTableModel, tea.Cmd) {
	l := m.container.Localizer
	cols := columnsForType(m.secretType, l)

	for i, row := range m.rows {
		for ci := range cols {
			zoneID := fmt.Sprintf("cell_%d_%d", i, ci)
			if !zone.Get(zoneID).InBounds(msg) {
				continue
			}

			// Binary-строка: последняя колонка — не payload (его нет, hasPayloadColumn==false),
			// а кликабельный текст «Скачать [D]», открывающий флоу скачивания с выбором папки.
			if domain.SecretType(row.Type) == domain.SecretTypeBinary && ci == len(cols)-1 {
				m.cursor = i
				return m, m.openDownloadPicker(i)
			}

			// Bank card: последняя колонка — отдельное действие «Показать [R]» (CVV/PIN/полный
			// номер), вместо обычного copy-on-click остальных колонок.
			if domain.SecretType(row.Type) == domain.SecretTypeBankCard && ci == len(cols)-1 {
				m.cursor = i
				return m, m.revealCard(row.ID)
			}

			// Определяем значение для копирования: если ячейка содержит маску (пароль/CVV/PAN
			// текущей строки ещё не раскрыт) — расшифровываем payload через команду.
			rowType := m.secretType
			revealedActive := i == m.cursor
			revealed := ""
			if revealedActive && m.revealedID == row.ID {
				revealed = m.revealedSecret
			}
			if m.secretType == domain.SecretTypeTOTP && m.totpRevealAll() {
				revealed = m.totpAllCodes[row.ID]
				revealedActive = revealed != ""
			}
			cells := rowCells(rowType, row, revealed, revealedActive, l)

			val := ""
			if ci < len(cells) {
				val = cells[ci]
			}

			// Если значение — маска, нужно расшифровать и скопировать реальное.
			if isMasked(val) {
				// Ставим курсор на эту строку и запускаем reveal+copy.
				m.cursor = i
				m.copiedRow, m.copiedCol, m.copiedVis = i, ci, true
				// После копирования запускаем reveal для этой строки, чтобы после сброса
				// copiedVis ячейка не оказалась пустой (раскрытый payload будет готов).
				return m, tea.Batch(m.revealAndCopy(row.ID, ci), m.copiedCellTimer(), m.scheduleReveal())
			}

			if val == "" {
				return m, nil
			}
			_ = clipboard.WriteAll(val)
			m.copiedRow, m.copiedCol, m.copiedVis = i, ci, true
			return m, m.copiedCellTimer()
		}
	}
	return m, nil
}

// copiedCellTimer запускает таймер, через 1.5с сбрасывающий визуальный индикатор ✓ в ячейке.
func (m dashboardTableModel) copiedCellTimer() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
		return copiedCellExpiredMsg{}
	})
}

// isMasked проверяет, является ли значение маской (только точки/пробелы).
func isMasked(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r != '•' && r != ' ' {
			return false
		}
	}
	return true
}

// revealAndCopy расшифровывает payload секрета и копирует значение указанной колонки.
func (m dashboardTableModel) revealAndCopy(secretID string, colIdx int) tea.Cmd {
	container := m.container
	ctx := m.ctx
	vaultID := m.vaultID
	rowType := m.secretTypeOf(secretID)

	return func() tea.Msg {
		var val string
		switch rowType {
		case domain.SecretTypeLoginPassword:
			d, err := container.Secret.GetDetail(ctx, vaultID, secretID)
			if err != nil {
				return payloadRevealErrMsg{err: err}
			}
			val = d.Payload.Password
		case domain.SecretTypeBankCard:
			d, err := container.Secret.GetBankCardDetail(ctx, vaultID, secretID)
			if err != nil {
				return payloadRevealErrMsg{err: err}
			}
			// Колонка 1 = Card/PAN — остальные колонки не маскированы.
			val = d.Payload.PAN
		case domain.SecretTypeTOTP:
			d, err := container.Secret.GetTOTPDetail(ctx, vaultID, secretID)
			if err != nil {
				return payloadRevealErrMsg{err: err}
			}
			code, _ := secret.GenerateTOTPCode(d.Payload)
			val = code
		default:
			return nil
		}
		if val != "" {
			_ = clipboard.WriteAll(val)
		}
		return nil
	}
}
