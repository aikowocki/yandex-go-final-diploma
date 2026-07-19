package tui

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

// formMode — текущий режим экрана секрета.
type formMode int

const (
	modeView           formMode = iota // read-only просмотр
	modeEdit                           // редактирование
	modeConfirmExit                    // подтверждение выхода с несохранёнными изменениями
	modeConfirmDelete                  // подтверждение удаления секрета
	modeFilePicker                     // выбор файла через filepicker (источник при загрузке)
	modeDownloadPicker                 // выбор папки назначения через filepicker (при скачивании)
)

// creatableTypes — типы секретов, создаваемые через форму.
var creatableTypes = []domain.SecretType{
	domain.SecretTypeLoginPassword,
	domain.SecretTypeText,
	domain.SecretTypeBankCard,
	domain.SecretTypeTOTP,
	domain.SecretTypeBinary,
}

type formFieldDef struct {
	key         string
	labelKey    string
	masked      bool
	multiline   bool // true → textarea вместо textinput
	placeholder string
	charLimit   int                 // 0 = default (512)
	validate    func(string) error  // nil = без валидации ввода
	autoFormat  func(string) string // nil = без автоформатирования (маски с разделителями)
}

func fieldsForType(t domain.SecretType) []formFieldDef {
	switch t {
	case domain.SecretTypeText:
		return []formFieldDef{
			{key: "title", labelKey: "tui_field_title"},
			{key: "body", labelKey: "tui_field_body", multiline: true},
			{key: "note", labelKey: "tui_field_note", multiline: true},
		}
	case domain.SecretTypeBankCard:
		return []formFieldDef{
			{key: "title", labelKey: "tui_field_title"},
			{key: "bank", labelKey: "tui_field_bank"},
			{key: "cardholder", labelKey: "tui_field_cardholder"},
			{key: "brand", labelKey: "tui_field_brand"},
			{key: "expiry", labelKey: "tui_field_expiry", placeholder: "MM/YY", charLimit: 5, validate: validateExpiry, autoFormat: formatExpiry},
			{key: "pan", labelKey: "tui_field_pan", placeholder: "1234 5678 9012 3456", charLimit: 19, validate: validatePAN, autoFormat: formatPAN},
			{key: "cvv", labelKey: "tui_field_cvv", masked: true, placeholder: "123", charLimit: 4, validate: validateDigits},
			{key: "pin", labelKey: "tui_field_pin", masked: true, placeholder: "1234", charLimit: 6, validate: validateDigits},
			{key: "note", labelKey: "tui_field_note", multiline: true},
		}
	case domain.SecretTypeTOTP:
		return []formFieldDef{
			{key: "title", labelKey: "tui_field_title"},
			{key: "issuer", labelKey: "tui_field_issuer"},
			{key: "account", labelKey: "tui_field_account"},
			{key: "secret", labelKey: "tui_field_totp_secret", masked: true, placeholder: "secret or otpauth:// URI"},
			{key: "note", labelKey: "tui_field_note", multiline: true},
		}
	case domain.SecretTypeBinary:
		return []formFieldDef{
			{key: "title", labelKey: "tui_field_title"},
			{key: "filepath", labelKey: "tui_field_filepath", placeholder: "/path/to/file (or Ctrl+B to browse)"},
			{key: "note", labelKey: "tui_field_note", multiline: true},
		}
	default:
		return []formFieldDef{
			{key: "title", labelKey: "tui_field_title"},
			{key: "uri", labelKey: "tui_field_uri"},
			{key: "username", labelKey: "tui_field_username"},
			{key: "password", labelKey: "tui_field_password", masked: true},
			{key: "note", labelKey: "tui_field_note", multiline: true},
		}
	}
}

type kvPair struct {
	key textinput.Model
	val textinput.Model
}

type otpItem struct {
	code textinput.Model
	used bool
}

type secretFormModel struct {
	ctx       context.Context
	container *app.Container

	vaultID   string
	vaultName string
	secretID  string
	version   int64

	secretType domain.SecretType
	mode       formMode
	defs       []formFieldDef
	inputs     []textinput.Model // для non-multiline полей (multiline → nil placeholder)
	textareas  []textarea.Model  // для multiline полей (non-multiline → zero-value placeholder)
	tags       textinput.Model
	custom     []kvPair
	otps       []otpItem

	focus  int // slot index
	err    error
	saving bool
	dirty  bool // были ли изменения в edit mode

	// filepicker — TUI file browser (Ctrl+B в поле filepath).
	filePicker filepicker.Model

	// snapshot — значения полей при входе в edit (для определения dirty).
	snapshot map[string]string
}

// --- Slots (навигация по полям) ---

type slotKind int

const (
	slotType slotKind = iota
	slotField
	slotTags
	slotCustomKey
	slotCustomVal
	slotAddCustom
	slotOTP
	slotAddOTP
)

type formSlot struct {
	kind slotKind
	idx  int
}

func (m secretFormModel) slots() []formSlot {
	var s []formSlot
	if m.typeSelectable() {
		s = append(s, formSlot{kind: slotType})
	}
	for i := range m.inputs {
		s = append(s, formSlot{kind: slotField, idx: i})
	}
	s = append(s, formSlot{kind: slotTags})
	for i := range m.custom {
		s = append(s, formSlot{kind: slotCustomKey, idx: i})
		s = append(s, formSlot{kind: slotCustomVal, idx: i})
	}
	if m.mode == modeEdit {
		s = append(s, formSlot{kind: slotAddCustom})
	}
	for i := range m.otps {
		s = append(s, formSlot{kind: slotOTP, idx: i})
	}
	if m.mode == modeEdit {
		s = append(s, formSlot{kind: slotAddOTP})
	}
	return s
}

func (m secretFormModel) currentSlot() (formSlot, bool) {
	sl := m.slots()
	if m.focus < 0 || m.focus >= len(sl) {
		return formSlot{}, false
	}
	return sl[m.focus], true
}

func (m secretFormModel) typeSelectable() bool { return m.secretID == "" && m.mode == modeEdit }

// --- Constructors ---

func newTextField(placeholder string, masked bool) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 512
	ti.Width = 50
	if masked {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}
	return ti
}

// newMultilineField создаёт «чистый» textarea для многострочных полей (body/note):
// без номеров строк, без фоновой подсветки строки курсора, с тонким левым рельсом.
func newMultilineField(placeholder string) textarea.Model {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.ShowLineNumbers = false
	ta.CharLimit = 4096
	// Без декоративного рельса "│": на пустых строках он тянется единой сплошной полосой
	// и не совпадает с отступом остальных полей формы (у них рельса нет вовсе). Двойной
	// пробел даёт тот же отступ "  ", что и у textinput.View().
	ta.Prompt = "  "
	ta.EndOfBufferCharacter = ' '
	ta.SetWidth(52)
	ta.SetHeight(3)

	// Убираем фоновую заливку строки курсора (дефолт даёт большой чёрный блок).
	focused, blurred := textarea.DefaultStyles()
	focused.CursorLine = lipgloss.NewStyle()
	focused.Prompt = lipgloss.NewStyle().Foreground(styles.Subtle)
	focused.Placeholder = lipgloss.NewStyle().Foreground(styles.Subtle)
	blurred.CursorLine = lipgloss.NewStyle()
	blurred.Prompt = lipgloss.NewStyle().Foreground(styles.Subtle)
	blurred.Placeholder = lipgloss.NewStyle().Foreground(styles.Subtle)
	ta.FocusedStyle = focused
	ta.BlurredStyle = blurred

	return ta
}

func newSecretFormModel(ctx context.Context, container *app.Container) secretFormModel {
	m := secretFormModel{ctx: ctx, container: container, secretType: domain.SecretTypeLoginPassword}
	m.tags = newTextField("", false)
	m = m.rebuildInputs()
	return m
}

func (m secretFormModel) rebuildInputs() secretFormModel {
	prev := map[string]string{}
	if m.defs != nil {
		for i, d := range m.defs {
			prev[d.key] = m.fieldValue(i)
		}
	}
	l := m.container.Localizer
	m.defs = fieldsForType(m.secretType)
	m.inputs = make([]textinput.Model, len(m.defs))
	m.textareas = make([]textarea.Model, len(m.defs))
	for i, d := range m.defs {
		ph := l.T(d.labelKey)
		if d.placeholder != "" {
			ph = d.placeholder
		}
		if d.multiline {
			m.textareas[i] = newMultilineField(ph)
			// Placeholder textinput (не используется для multiline, но слот нужен).
			m.inputs[i] = textinput.New()
		} else {
			limit := 512
			if d.charLimit > 0 {
				limit = d.charLimit
			}
			ti := newTextField(ph, d.masked)
			ti.CharLimit = limit
			if d.validate != nil {
				fn := d.validate
				ti.Validate = func(s string) error { return fn(s) }
			}
			m.inputs[i] = ti
		}
		if v, ok := prev[d.key]; ok {
			m.setFieldValue(i, v)
		}
	}
	m.applyFocus()
	return m
}

// isMultiline сообщает, является ли поле по индексу многострочным (textarea).
func (m secretFormModel) isMultiline(idx int) bool {
	return idx >= 0 && idx < len(m.defs) && m.defs[idx].multiline
}

// fieldValue возвращает значение поля по индексу (textinput или textarea).
func (m secretFormModel) fieldValue(idx int) string {
	if idx < len(m.defs) && m.defs[idx].multiline {
		return m.textareas[idx].Value()
	}
	if idx < len(m.inputs) {
		return m.inputs[idx].Value()
	}
	return ""
}

// setFieldValue устанавливает значение поля по индексу.
func (m secretFormModel) setFieldValue(idx int, val string) secretFormModel {
	if idx < len(m.defs) && m.defs[idx].multiline {
		m.textareas[idx].SetValue(val)
	} else if idx < len(m.inputs) {
		m.inputs[idx].SetValue(val)
	}
	return m
}

func (m secretFormModel) applyFocus() secretFormModel {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	for i := range m.textareas {
		m.textareas[i].Blur()
	}
	m.tags.Blur()
	for i := range m.custom {
		m.custom[i].key.Blur()
		m.custom[i].val.Blur()
	}
	for i := range m.otps {
		m.otps[i].code.Blur()
	}
	if m.mode != modeEdit {
		return m // в view mode инпуты не фокусируются
	}
	sl := m.slots()
	if m.focus < 0 || m.focus >= len(sl) {
		return m
	}
	switch s := sl[m.focus]; s.kind {
	case slotField:
		if s.idx < len(m.defs) && m.defs[s.idx].multiline {
			m.textareas[s.idx].Focus()
		} else {
			m.inputs[s.idx].Focus()
		}
	case slotTags:
		m.tags.Focus()
	case slotCustomKey:
		m.custom[s.idx].key.Focus()
	case slotCustomVal:
		m.custom[s.idx].val.Focus()
	case slotOTP:
		m.otps[s.idx].code.Focus()
	}
	return m
}

// --- SetCreate / SetEditData ---

func (m secretFormModel) reset(vaultID, vaultName string) secretFormModel {
	m.vaultID = vaultID
	m.vaultName = vaultName
	m.err = nil
	m.saving = false
	m.dirty = false
	m.snapshot = nil
	m.defs = nil
	m.inputs = nil
	m.tags = newTextField("", false)
	m.custom = nil
	m.otps = nil
	return m
}

func (m secretFormModel) SetCreate(vaultID, vaultName string, preselect domain.SecretType) secretFormModel {
	m = m.reset(vaultID, vaultName)
	m.secretID = ""
	m.version = 0
	m.mode = modeEdit
	m.focus = 0
	if !isCreatableType(preselect) {
		preselect = domain.SecretTypeLoginPassword
	}
	m.secretType = preselect
	m = m.rebuildInputs()
	return m
}

func (m secretFormModel) SetEditData(vaultID, vaultName, secretID string, version int64, t domain.SecretType, data formData) secretFormModel {
	return m.SetEditDataWithMode(vaultID, vaultName, secretID, version, t, data, false)
}

// SetEditDataWithMode — то же, что SetEditData, но с явным контролем режима открытия.
// startInEdit=true — сразу edit mode (шорткат [e] в таблице), иначе обычный read-only view.
func (m secretFormModel) SetEditDataWithMode(vaultID, vaultName, secretID string, version int64, t domain.SecretType, data formData, startInEdit bool) secretFormModel {
	m = m.reset(vaultID, vaultName)
	m.secretID = secretID
	m.version = version
	m.secretType = t
	m.mode = modeView
	m.focus = 0
	m = m.rebuildInputs()
	for k, v := range data.fields {
		m = m.setValue(k, v)
	}
	m.tags.SetValue(data.tags)
	for _, kv := range data.custom {
		m.custom = append(m.custom, m.newKVPair(kv.Key, kv.Value))
	}
	for _, oc := range data.otpCodes {
		m.otps = append(m.otps, m.newOTPItem(oc.Code, oc.Used))
	}
	m = m.takeSnapshot()
	if startInEdit {
		m.mode = modeEdit
		m = m.applyFocus()
	}
	return m
}

func (m secretFormModel) takeSnapshot() secretFormModel {
	m.snapshot = map[string]string{}
	for _, d := range m.defs {
		m.snapshot[d.key] = m.value(d.key)
	}
	m.snapshot["__tags"] = m.tags.Value()
	m.dirty = false
	return m
}

func (m secretFormModel) isDirty() bool {
	if m.dirty {
		return true
	}
	if m.snapshot == nil {
		return false
	}
	for _, d := range m.defs {
		if m.value(d.key) != m.snapshot[d.key] {
			return true
		}
	}
	return m.tags.Value() != m.snapshot["__tags"]
}

func (m secretFormModel) setValue(key, val string) secretFormModel {
	for i, d := range m.defs {
		if d.key == key {
			return m.setFieldValue(i, val)
		}
	}
	return m
}

func (m secretFormModel) value(key string) string {
	for i, d := range m.defs {
		if d.key == key {
			return m.fieldValue(i)
		}
	}
	return ""
}

func (m secretFormModel) newKVPair(k, v string) kvPair {
	kp := newTextField(m.container.Localizer.T("tui_field_cf_key"), false)
	vp := newTextField(m.container.Localizer.T("tui_field_cf_value"), false)
	kp.SetValue(k)
	vp.SetValue(v)
	return kvPair{key: kp, val: vp}
}

func (m secretFormModel) newOTPItem(code string, used bool) otpItem {
	c := newTextField(m.container.Localizer.T("tui_field_otp_code"), false)
	c.SetValue(code)
	return otpItem{code: c, used: used}
}

func isCreatableType(t domain.SecretType) bool {
	for _, c := range creatableTypes {
		if c == t {
			return true
		}
	}
	return false
}

func (m secretFormModel) Init() tea.Cmd { return textinput.Blink }

// --- Update ---

func (m secretFormModel) update(msg tea.Msg) (secretFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.mode {
		case modeConfirmExit:
			return m.handleConfirmExit(msg)
		case modeConfirmDelete:
			return m.handleConfirmDelete(msg)
		case modeView:
			return m.handleViewKey(msg)
		case modeEdit:
			return m.handleEditKey(msg)
		case modeFilePicker:
			return m.handleFilePickerKey(msg)
		case modeDownloadPicker:
			return m.handleDownloadPickerKey(msg)
		}
	case loginErrMsg:
		m.saving = false
		m.err = msg.err
		return m, nil
	}

	if m.mode == modeFilePicker {
		return m.updateFilePicker(msg)
	}
	if m.mode == modeDownloadPicker {
		return m.updateDownloadPicker(msg)
	}
	if m.mode == modeEdit {
		return m.updateActiveInput(msg)
	}
	return m, nil
}

// --- View mode keys ---

func (m secretFormModel) handleViewKey(msg tea.KeyMsg) (secretFormModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		return m, m.backToDashboard()
	case tea.KeyUp:
		m = m.focusPrev()
		return m, nil
	case tea.KeyDown:
		m = m.focusNext()
		return m, nil
	case tea.KeySpace, tea.KeyCtrlU:
		// Toggle OTP used по space или Ctrl+U. Остаёмся в view mode (не вызываем save —
		// toggle сохраняется через отдельную команду, которая не уходит на dashboard).
		if s, ok := m.currentSlot(); ok && s.kind == slotOTP {
			m.otps[s.idx].used = !m.otps[s.idx].used
			m.dirty = true
			return m, m.saveOTPToggle()
		}
		return m, nil
	case tea.KeyDelete, tea.KeyBackspace:
		// [Del/Backspace] — удалить секрет (с подтверждением через confirm mode).
		if m.secretID != "" {
			m.mode = modeConfirmDelete
			return m, nil
		}
		return m, nil
	}
	if isShortcut(msg, "c") {
		return m, m.copyCurrentField()
	}
	if isShortcut(msg, "d") {
		// [D] — для binary: открыть флоу скачивания (выбор папки назначения); для остальных: удалить.
		if m.secretType == domain.SecretTypeBinary && m.secretID != "" {
			home, _ := os.UserHomeDir()
			fp := filepicker.New()
			fp.CurrentDirectory = home
			fp.ShowHidden = false
			fp.DirAllowed = true
			fp.FileAllowed = false
			fp.SetHeight(15)
			m.filePicker = fp
			m.mode = modeDownloadPicker
			return m, m.filePicker.Init()
		}
		if m.secretID != "" {
			m.mode = modeConfirmDelete
			return m, nil
		}
	}
	if isShortcut(msg, "e") {
		m.mode = modeEdit
		m = m.takeSnapshot()
		m = m.applyFocus()
		return m, nil
	}
	return m, nil
}

// --- Edit mode keys ---

func (m secretFormModel) handleEditKey(msg tea.KeyMsg) (secretFormModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		if m.isDirty() {
			m.mode = modeConfirmExit
			return m, nil
		}
		if m.secretID == "" {
			return m, m.backToDashboard() // создание — назад на dashboard
		}
		m.mode = modeView
		m = m.applyFocus()
		return m, nil
	case tea.KeyTab:
		m = m.focusNext()
		return m, nil
	case tea.KeyShiftTab:
		m = m.focusPrev()
		return m, nil
	case tea.KeyDown:
		// В multiline-поле стрелки двигают курсор внутри текста, а не между полями.
		if s, ok := m.currentSlot(); ok && s.kind == slotField && m.isMultiline(s.idx) {
			return m.updateActiveInput(msg)
		}
		m = m.focusNext()
		return m, nil
	case tea.KeyUp:
		if s, ok := m.currentSlot(); ok && s.kind == slotField && m.isMultiline(s.idx) {
			return m.updateActiveInput(msg)
		}
		m = m.focusPrev()
		return m, nil
	case tea.KeyCtrlS:
		m, cmd := m.save()
		return m, cmd
	case tea.KeyCtrlA:
		m.custom = append(m.custom, m.newKVPair("", ""))
		m.dirty = true
		m = m.focusSlot(slotCustomKey, len(m.custom)-1)
		return m, nil
	case tea.KeyCtrlO:
		m.otps = append(m.otps, m.newOTPItem("", false))
		m.dirty = true
		m = m.focusSlot(slotOTP, len(m.otps)-1)
		return m, nil
	case tea.KeyCtrlB:
		// Ctrl+B — открыть file picker (только для binary-типа с полем filepath).
		if m.secretType == domain.SecretTypeBinary {
			home, _ := os.UserHomeDir()
			fp := filepicker.New()
			fp.CurrentDirectory = home
			fp.ShowHidden = false
			fp.SetHeight(15)
			m.filePicker = fp
			m.mode = modeFilePicker
			return m, m.filePicker.Init()
		}
		return m, nil
	case tea.KeyCtrlU:
		if s, ok := m.currentSlot(); ok && s.kind == slotOTP {
			m.otps[s.idx].used = !m.otps[s.idx].used
			m.dirty = true
		}
		return m, nil
	case tea.KeyEnter:
		if s, ok := m.currentSlot(); ok {
			switch s.kind {
			case slotAddCustom:
				m.custom = append(m.custom, m.newKVPair("", ""))
				m.dirty = true
				m = m.focusSlot(slotCustomKey, len(m.custom)-1)
				return m, nil
			case slotAddOTP:
				m.otps = append(m.otps, m.newOTPItem("", false))
				m.dirty = true
				m = m.focusSlot(slotOTP, len(m.otps)-1)
				return m, nil
			case slotType:
				return m, nil
			case slotField:
				// Multiline поля обрабатывают Enter сами (новая строка).
				if s.idx < len(m.defs) && m.defs[s.idx].multiline {
					return m.updateActiveInput(msg)
				}
			}
		}
		m = m.focusNext()
		return m, nil
	case tea.KeyLeft:
		if s, ok := m.currentSlot(); ok && s.kind == slotType {
			m = m.cycleType(-1)
			return m, nil
		}
	case tea.KeyRight:
		if s, ok := m.currentSlot(); ok && s.kind == slotType {
			m = m.cycleType(1)
			return m, nil
		}
	}

	return m.updateActiveInput(msg)
}

// --- Confirm exit ---

func (m secretFormModel) handleConfirmExit(msg tea.KeyMsg) (secretFormModel, tea.Cmd) {
	// Сверяем через isShortcut/normalizeKey, а не msg.String() напрямую — иначе "y"/"n" не
	// сработают в русской раскладке (физическая клавиша печатает "н"/"т").
	switch {
	case isShortcut(msg, "y"):
		m.mode = modeEdit
		m, cmd := m.save()
		return m, cmd
	case isShortcut(msg, "n"):
		m.dirty = false
		if m.secretID == "" {
			return m, m.backToDashboard()
		}
		m.mode = modeView
		m = m.applyFocus()
		return m, nil
	case msg.Type == tea.KeyEsc:
		m.mode = modeEdit
		return m, nil
	}
	return m, nil
}

// --- Confirm delete ---

func (m secretFormModel) handleConfirmDelete(msg tea.KeyMsg) (secretFormModel, tea.Cmd) {
	switch {
	case isShortcut(msg, "y"):
		return m, m.deleteSecret()
	case isShortcut(msg, "n"), msg.Type == tea.KeyEsc:
		m.mode = modeView
		return m, nil
	}
	return m, nil
}

// deleteSecret удаляет секрет через usecase и возвращает на dashboard. При конфликте версий
// (кто-то изменил секрет на другом устройстве после того, как мы прочитали baseVersion)
// ведёт на общий экран разрешения конфликтов вместо тихого игнорирования.
func (m secretFormModel) deleteSecret() tea.Cmd {
	container := m.container
	ctx := m.ctx
	vaultID := m.vaultID
	vaultName := m.vaultName
	secretID := m.secretID
	version := m.version
	return func() tea.Msg {
		conflict, err := container.Secret.DeleteSecret(ctx, vaultID, secretID, version)
		if err != nil {
			slog.Error("delete secret failed", "err", err)
			return loginErrMsg{err: err}
		}
		if conflict != nil {
			return switchScreenMsg{screen: screenConflict, conflict: conflict, vaultID: vaultID, vaultName: vaultName}
		}
		return switchScreenMsg{screen: screenDashboard, vaultID: vaultID, vaultName: vaultName}
	}
}

// saveOTPToggle сохраняет секрет (все поля включая обновлённые OTP-коды) но остаётся в view mode.
func (m secretFormModel) saveOTPToggle() tea.Cmd {
	container := m.container
	ctx := m.ctx
	vaultID := m.vaultID
	secretID := m.secretID
	version := m.version
	secretType := m.secretType

	vals := map[string]string{}
	for _, d := range m.defs {
		vals[d.key] = m.value(d.key)
	}
	tags, custom, otp := m.collectExtras()

	return func() tea.Msg {
		var err error
		switch secretType {
		case domain.SecretTypeText:
			in := secret.CreateTextInput{Title: vals["title"], Tags: tags, Note: vals["note"], CustomFields: custom, Body: vals["body"], OTPCodes: otp}
			_, err = container.Secret.UpdateText(ctx, vaultID, secretID, version, in)
		case domain.SecretTypeBankCard:
			in := secret.CreateBankCardInput{Title: vals["title"], Tags: tags, Bank: vals["bank"], Cardholder: vals["cardholder"], Brand: vals["brand"], Expiry: vals["expiry"], Note: vals["note"], CustomFields: custom, PAN: vals["pan"], CVV: vals["cvv"], PIN: vals["pin"], OTPCodes: otp}
			_, err = container.Secret.UpdateBankCard(ctx, vaultID, secretID, version, in)
		case domain.SecretTypeTOTP:
			in := secret.CreateTOTPInput{Title: vals["title"], Tags: tags, Issuer: vals["issuer"], Account: vals["account"], Note: vals["note"], CustomFields: custom, Secret: vals["secret"], OTPCodes: otp}
			_, err = container.Secret.UpdateTOTP(ctx, vaultID, secretID, version, in)
		default:
			in := secret.CreateLoginPasswordInput{Title: vals["title"], Tags: tags, URI: vals["uri"], Username: vals["username"], Note: vals["note"], CustomFields: custom, Password: vals["password"], OTPCodes: otp}
			_, err = container.Secret.UpdateLoginPassword(ctx, vaultID, secretID, version, in)
		}
		if err != nil {
			return loginErrMsg{err: err}
		}
		return toastMsg{text: "✓"}
	}
}

// --- Copy current field ---

func (m secretFormModel) copyCurrentField() tea.Cmd {
	s, ok := m.currentSlot()
	if !ok {
		return nil
	}
	var val string
	switch s.kind {
	case slotField:
		val = m.fieldValue(s.idx)
	case slotTags:
		val = m.tags.Value()
	case slotCustomKey:
		val = m.custom[s.idx].key.Value()
	case slotCustomVal:
		val = m.custom[s.idx].val.Value()
	case slotOTP:
		val = m.otps[s.idx].code.Value()
	}
	if val == "" {
		return nil
	}
	_ = clipboard.WriteAll(val)
	return showToast(m.container.Localizer.T("tui_toast_copied"))
}

// --- Navigation helpers ---

func (m secretFormModel) focusNext() secretFormModel {
	n := len(m.slots())
	if n == 0 {
		return m
	}
	m.focus = (m.focus + 1) % n
	if s, _ := m.currentSlot(); s.kind == slotType && !m.typeSelectable() {
		m.focus = (m.focus + 1) % n
	}
	return m.applyFocus()
}

func (m secretFormModel) focusPrev() secretFormModel {
	n := len(m.slots())
	if n == 0 {
		return m
	}
	m.focus = (m.focus - 1 + n) % n
	if s, _ := m.currentSlot(); s.kind == slotType && !m.typeSelectable() {
		m.focus = (m.focus - 1 + n) % n
	}
	return m.applyFocus()
}

func (m secretFormModel) focusSlot(kind slotKind, idx int) secretFormModel {
	for i, s := range m.slots() {
		if s.kind == kind && s.idx == idx {
			m.focus = i
			return m.applyFocus()
		}
	}
	return m
}

func (m secretFormModel) cycleType(dir int) secretFormModel {
	idx := 0
	for i, t := range creatableTypes {
		if t == m.secretType {
			idx = i
			break
		}
	}
	n := len(creatableTypes)
	m.secretType = creatableTypes[(idx+dir+n)%n]
	return m.rebuildInputs()
}

func (m secretFormModel) updateActiveInput(msg tea.Msg) (secretFormModel, tea.Cmd) {
	s, ok := m.currentSlot()
	if !ok {
		return m, nil
	}
	// Снимок значения текущего поля до обновления — чтобы отличить реальное изменение
	// текста от навигационных клавиш (←/→/Home/End), которые просто двигают курсор
	// и не должны помечать форму как "изменённую" (dirty), иначе Esc спрашивает
	// подтверждение выхода даже когда пользователь ничего не поменял.
	before := m.currentSlotValue(s)
	var cmd tea.Cmd
	switch s.kind {
	case slotField:
		if s.idx < len(m.defs) && m.defs[s.idx].multiline {
			m.textareas[s.idx], cmd = m.textareas[s.idx].Update(msg)
		} else {
			m.inputs[s.idx], cmd = m.inputs[s.idx].Update(msg)
		}
		// Автоформатирование (маска с разделителями): после каждого ввода трансформируем
		// значение (например "1234" → "12/34" для expiry, "12345678" → "1234 5678" для PAN).
		if s.idx < len(m.defs) && m.defs[s.idx].autoFormat != nil {
			old := m.inputs[s.idx].Value()
			formatted := m.defs[s.idx].autoFormat(old)
			if formatted != old {
				m.inputs[s.idx].SetValue(formatted)
				m.inputs[s.idx].CursorEnd()
			}
		}
		// Автоопределение платёжной системы по BIN при вводе PAN (карточка).
		if s.idx < len(m.defs) && m.defs[s.idx].key == "pan" && m.secretType == domain.SecretTypeBankCard {
			digits := onlyDigits(m.inputs[s.idx].Value())
			if brand := detectCardBrand(digits); brand != "" {
				m = m.setValue("brand", brand)
			}
		}
		// Автообработка file:// URI в поле filepath.
		if s.idx < len(m.defs) && m.defs[s.idx].key == "filepath" {
			val := m.inputs[s.idx].Value()
			if strings.HasPrefix(strings.TrimSpace(val), "file://") {
				resolved := resolveFilePath(val)
				m.inputs[s.idx].SetValue(resolved)
				m.inputs[s.idx].CursorEnd()
			}
		}
		// Автопарсинг otpauth:// URI: если пользователь вставил URI в поле «secret» TOTP-формы,
		// автоматически распарсить и заполнить issuer/account/title.
		if s.idx < len(m.defs) && m.defs[s.idx].key == "secret" && m.secretType == domain.SecretTypeTOTP {
			val := strings.TrimSpace(m.inputs[s.idx].Value())
			if strings.HasPrefix(val, "otpauth://") {
				if parsed, err := secret.ParseOTPAuthURI(val); err == nil {
					m = m.setValue("secret", parsed.Secret)
					if parsed.Issuer != "" {
						m = m.setValue("issuer", parsed.Issuer)
					}
					if parsed.Account != "" {
						m = m.setValue("account", parsed.Account)
					}
					if parsed.Title != "" && m.value("title") == "" {
						m = m.setValue("title", parsed.Title)
					}
					cmd = showToast(m.container.Localizer.T("tui_toast_uri_parsed"))
				}
			}
		}
	case slotTags:
		m.tags, cmd = m.tags.Update(msg)
	case slotCustomKey:
		m.custom[s.idx].key, cmd = m.custom[s.idx].key.Update(msg)
	case slotCustomVal:
		m.custom[s.idx].val, cmd = m.custom[s.idx].val.Update(msg)
	case slotOTP:
		m.otps[s.idx].code, cmd = m.otps[s.idx].code.Update(msg)
	}
	if m.currentSlotValue(s) != before {
		m.dirty = true
	}
	return m, cmd
}

// currentSlotValue возвращает текущее текстовое значение поля указанного слота
// (используется для сравнения до/после Update, чтобы отличить реальный ввод от
// чисто навигационных клавиш курсора).
func (m secretFormModel) currentSlotValue(s formSlot) string {
	switch s.kind {
	case slotField:
		return m.fieldValue(s.idx)
	case slotTags:
		return m.tags.Value()
	case slotCustomKey:
		return m.custom[s.idx].key.Value()
	case slotCustomVal:
		return m.custom[s.idx].val.Value()
	case slotOTP:
		return m.otps[s.idx].code.Value()
	}
	return ""
}

func (m secretFormModel) backToDashboard() tea.Cmd {
	vaultID, vaultName := m.vaultID, m.vaultName
	return func() tea.Msg {
		return switchScreenMsg{screen: screenDashboard, vaultID: vaultID, vaultName: vaultName}
	}
}

// --- Save ---

func (m secretFormModel) collectExtras() (tags []string, custom []secretcontent.KeyValue, otp []secretcontent.OTPCode) {
	for _, t := range strings.Split(m.tags.Value(), ",") {
		if s := strings.TrimSpace(t); s != "" {
			tags = append(tags, s)
		}
	}
	for _, c := range m.custom {
		k := strings.TrimSpace(c.key.Value())
		if k == "" {
			continue
		}
		custom = append(custom, secretcontent.KeyValue{Key: k, Value: c.val.Value()})
	}
	for _, o := range m.otps {
		code := strings.TrimSpace(o.code.Value())
		if code == "" {
			continue
		}
		otp = append(otp, secretcontent.OTPCode{Code: code, Used: o.used})
	}
	return
}

// save валидирует форму и возвращает обновлённую модель (с выставленными err/saving)
// вместе с командой сохранения (может быть nil при ошибке валидации).
func (m secretFormModel) save() (secretFormModel, tea.Cmd) {
	title := m.value("title")
	if title == "" {
		m.err = fmt.Errorf("%s", m.container.Localizer.T("tui_err_title_required"))
		return m, nil
	}
	// Luhn-валидация PAN для банковских карт (если заполнен).
	if m.secretType == domain.SecretTypeBankCard {
		pan := onlyDigits(m.value("pan"))
		if len(pan) >= 12 && !luhnValid(pan) {
			m.err = fmt.Errorf("%s", m.container.Localizer.T("tui_err_luhn_invalid"))
			return m, nil
		}
	}
	// Уникальность ключей custom fields.
	if dupKey := findDuplicateCustomKey(m.custom); dupKey != "" {
		m.err = fmt.Errorf("%s: %s", m.container.Localizer.T("tui_err_duplicate_custom_key"), dupKey)
		return m, nil
	}
	// Уникальность OTP recovery codes.
	if dupCode := findDuplicateOTPCode(m.otps); dupCode != "" {
		m.err = fmt.Errorf("%s: %s", m.container.Localizer.T("tui_err_duplicate_otp_code"), dupCode)
		return m, nil
	}
	m.saving = true
	m.err = nil

	container := m.container
	ctx := m.ctx
	vaultID := m.vaultID
	vaultName := m.vaultName
	secretID := m.secretID
	version := m.version
	secretType := m.secretType

	vals := map[string]string{}
	for _, d := range m.defs {
		vals[d.key] = m.value(d.key)
	}
	tags, custom, otp := m.collectExtras()

	back := switchScreenMsg{screen: screenDashboard, vaultID: vaultID, vaultName: vaultName}
	toGenericConflict := func(c *secret.GenericConflict) tea.Msg {
		return switchScreenMsg{screen: screenConflict, conflict: c, vaultID: vaultID, vaultName: vaultName}
	}

	return m, func() tea.Msg {
		switch secretType {
		case domain.SecretTypeText:
			in := secret.CreateTextInput{Title: vals["title"], Tags: tags, Note: vals["note"], CustomFields: custom, Body: vals["body"], OTPCodes: otp}
			if secretID == "" {
				_, err := container.Secret.CreateText(ctx, vaultID, in)
				return doneOrErr(err, back)
			}
			c, err := container.Secret.UpdateText(ctx, vaultID, secretID, version, in)
			return genericResult(c, err, toGenericConflict, back)
		case domain.SecretTypeBankCard:
			in := secret.CreateBankCardInput{Title: vals["title"], Tags: tags, Bank: vals["bank"], Cardholder: vals["cardholder"], Brand: vals["brand"], Expiry: vals["expiry"], Note: vals["note"], CustomFields: custom, PAN: vals["pan"], CVV: vals["cvv"], PIN: vals["pin"], OTPCodes: otp}
			if secretID == "" {
				_, err := container.Secret.CreateBankCard(ctx, vaultID, in)
				return doneOrErr(err, back)
			}
			c, err := container.Secret.UpdateBankCard(ctx, vaultID, secretID, version, in)
			return genericResult(c, err, toGenericConflict, back)
		case domain.SecretTypeTOTP:
			in := secret.CreateTOTPInput{Title: vals["title"], Tags: tags, Issuer: vals["issuer"], Account: vals["account"], Note: vals["note"], CustomFields: custom, Secret: vals["secret"], OTPCodes: otp}
			if secretID == "" {
				_, err := container.Secret.CreateTOTP(ctx, vaultID, in)
				return doneOrErr(err, back)
			}
			c, err := container.Secret.UpdateTOTP(ctx, vaultID, secretID, version, in)
			return genericResult(c, err, toGenericConflict, back)
		case domain.SecretTypeBinary:
			fp := resolveFilePath(vals["filepath"])
			if secretID == "" {
				// Создание: читаем файл с диска.
				f, err := os.Open(fp)
				if err != nil {
					return loginErrMsg{err: fmt.Errorf("open file: %w", err)}
				}
				defer func() { _ = f.Close() }()
				info, err := f.Stat()
				if err != nil {
					return loginErrMsg{err: fmt.Errorf("stat file: %w", err)}
				}
				in := secret.CreateBinaryInput{
					Title:        vals["title"],
					Tags:         tags,
					Filename:     filepath.Base(fp),
					Note:         vals["note"],
					CustomFields: custom,
					Data:         f,
					Size:         info.Size(),
					OTPCodes:     otp,
				}
				_, err = container.Secret.CreateBinary(ctx, vaultID, in)
				return doneOrErr(err, back)
			}
			// Обновление binary: пока не поддерживается (нет usecase UpdateBinary).
			return back
		default:
			in := secret.CreateLoginPasswordInput{Title: vals["title"], Tags: tags, URI: vals["uri"], Username: vals["username"], Note: vals["note"], CustomFields: custom, Password: vals["password"], OTPCodes: otp}
			if secretID == "" {
				_, err := container.Secret.CreateLoginPassword(ctx, vaultID, in)
				return doneOrErr(err, back)
			}
			c, err := container.Secret.UpdateLoginPassword(ctx, vaultID, secretID, version, in)
			return genericResult(c, err, toGenericConflict, back)
		}
	}
}

func doneOrErr(err error, back switchScreenMsg) tea.Msg {
	if err != nil {
		slog.Error("secret save failed", "err", err)
		return loginErrMsg{err: err}
	}
	return back
}

func genericResult(c *secret.GenericConflict, err error, toGenericConflict func(*secret.GenericConflict) tea.Msg, back switchScreenMsg) tea.Msg {
	if err != nil {
		slog.Error("secret save failed", "err", err)
		return loginErrMsg{err: err}
	}
	if c != nil {
		return toGenericConflict(c)
	}
	return back
}

func typeLabel(t domain.SecretType, l localizerT) string {
	switch t {
	case domain.SecretTypeText:
		return l.T("tui_tab_notes")
	case domain.SecretTypeBankCard:
		return l.T("tui_tab_cards")
	case domain.SecretTypeTOTP:
		return l.T("tui_tab_totp")
	case domain.SecretTypeBinary:
		return l.T("tui_tab_files")
	default:
		return l.T("tui_tab_logins")
	}
}

// --- View ---

func (m secretFormModel) view(width, height int) string {
	content := m.renderContent()
	lines := strings.Split(content, "\n")
	if height <= 0 || len(lines) <= height {
		return content
	}
	focusLine := 0
	for i, ln := range lines {
		if strings.Contains(ln, "▸") {
			focusLine = i
			break
		}
	}
	margin := height / 3
	start := focusLine - margin
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
		start = end - height
		if start < 0 {
			start = 0
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func (m secretFormModel) renderContent() string {
	var b strings.Builder
	l := m.container.Localizer
	cur, _ := m.currentSlot()

	// Заголовок.
	switch {
	case m.mode == modeFilePicker:
		b.WriteString(styles.Title.Render("📂 " + l.T("tui_filepicker_title")))
		b.WriteString("\n\n")
		b.WriteString(m.filePicker.View())
		b.WriteString("\n\n")
		b.WriteString(styles.HelpText.Render(l.T("tui_help_filepicker")))
		return b.String()
	case m.mode == modeDownloadPicker:
		b.WriteString(styles.Title.Render("📂 " + l.T("tui_download_picker_title")))
		b.WriteString("\n\n")
		b.WriteString(m.filePicker.View())
		b.WriteString("\n\n")
		b.WriteString(styles.HelpText.Render(l.T("tui_help_download_picker")))
		return b.String()
	case m.mode == modeConfirmExit:
		b.WriteString(styles.Title.Render("⚠️  " + l.T("tui_confirm_exit_title")))
		b.WriteString("\n\n")
		b.WriteString(styles.InputLabel.Render(l.T("tui_confirm_exit_prompt")))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpText.Render(l.T("tui_help_confirm_exit")))
		return b.String()
	case m.mode == modeConfirmDelete:
		b.WriteString(styles.Title.Render("⚠️  " + l.T("tui_confirm_delete_title")))
		b.WriteString("\n\n")
		b.WriteString(styles.ErrorText.Render(l.T("tui_confirm_delete_prompt")))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpText.Render(l.T("tui_help_confirm_delete")))
		return b.String()
	case m.secretID == "" && m.mode == modeEdit:
		b.WriteString(styles.Title.Render("✏️  " + l.T("tui_create_secret_title")))
	case m.mode == modeEdit:
		b.WriteString(styles.Title.Render("✏️  " + l.T("tui_edit_secret")))
	default:
		b.WriteString(styles.Title.Render("📋 " + l.T("tui_detail_title")))
	}
	b.WriteString("\n\n")

	// Селектор типа (только при создании в edit mode).
	if m.typeSelectable() {
		typeVal := "‹ " + typeLabel(m.secretType, l) + " ›"
		line := fmt.Sprintf("%s: %s", l.T("tui_field_type"), typeVal)
		b.WriteString(cursorLine(cur.kind == slotType, line))
		b.WriteString("\n\n")
	}

	// Поля.
	for i, d := range m.defs {
		focused := cur.kind == slotField && cur.idx == i
		label := l.T(d.labelKey)
		if m.mode == modeView {
			val := m.fieldValue(i)
			if d.masked && val != "" {
				val = strings.Repeat("•", min(len(val), 12)) // маскируем в view
			}
			if d.multiline && val != "" {
				// Многострочное значение: метка на своей строке, текст с ручным
				// word-wrap и постоянным отступом. Раньше короткий текст без явных
				// "\n" уходил в viewFieldLine одной длинной строкой, которую
				// терминал сам заворачивал без отступа — из-за этого перенесённые
				// строки "съезжали" к левому краю и текст казался неровным.
				mark := "  "
				if focused {
					mark = "▸ "
				}
				b.WriteString(mark + styles.InputLabel.Render(label) + ":")
				for _, srcLn := range strings.Split(val, "\n") {
					// TrimSpace убирает случайные пробелы в начале строки (например,
					// если пользователь напечатал "  2 строка" вместо "2 строка") —
					// иначе такие строки визуально "съезжают" относительно соседних
					// с одинаковым отступом "    ".
					for _, ln := range wrapText(strings.TrimSpace(srcLn), 60) {
						b.WriteString("\n    " + ln)
					}
				}
			} else {
				b.WriteString(viewFieldLine(focused, label, val))
			}
		} else {
			b.WriteString(labelLine(focused, label))
			b.WriteString("\n")
			if d.multiline {
				b.WriteString(m.textareas[i].View())
			} else {
				b.WriteString("  ")
				b.WriteString(m.inputs[i].View())
			}
		}
		b.WriteString("\n\n")
	}

	// Tags.
	if m.mode == modeView {
		b.WriteString(viewFieldLine(cur.kind == slotTags, l.T("tui_field_tags"), m.tags.Value()))
	} else {
		b.WriteString(labelLine(cur.kind == slotTags, l.T("tui_field_tags")))
		b.WriteString("\n  ")
		b.WriteString(m.tags.View())
	}
	b.WriteString("\n\n")

	// Custom fields.
	if len(m.custom) > 0 || m.mode == modeEdit {
		b.WriteString(styles.Subtitle.Render(l.T("tui_field_custom")))
		b.WriteString("\n")
		for i := range m.custom {
			keyFocused := cur.kind == slotCustomKey && cur.idx == i
			valFocused := cur.kind == slotCustomVal && cur.idx == i
			mark := "  "
			if keyFocused || valFocused {
				mark = "▸ "
			}
			if m.mode == modeView {
				_, _ = fmt.Fprintf(&b, "%s%s = %s", mark, m.custom[i].key.Value(), m.custom[i].val.Value())
			} else {
				b.WriteString(mark)
				b.WriteString(m.custom[i].key.View())
				b.WriteString("  =  ")
				b.WriteString(m.custom[i].val.View())
			}
			b.WriteString("\n")
		}
		if m.mode == modeEdit {
			b.WriteString(cursorLine(cur.kind == slotAddCustom, "[+ "+l.T("tui_field_add_custom")+"]"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// OTP codes.
	if len(m.otps) > 0 || m.mode == modeEdit {
		b.WriteString(styles.Subtitle.Render(l.T("tui_field_otp_codes")))
		b.WriteString("\n")
		for i := range m.otps {
			focused := cur.kind == slotOTP && cur.idx == i
			box := "[ ]"
			if m.otps[i].used {
				box = "[x]"
			}
			mark := "  "
			if focused {
				mark = "▸ "
			}
			if m.mode == modeView {
				_, _ = fmt.Fprintf(&b, "%s%s %s", mark, box, m.otps[i].code.Value())
			} else {
				b.WriteString(mark + box + " ")
				b.WriteString(m.otps[i].code.View())
			}
			b.WriteString("\n")
		}
		if m.mode == modeEdit {
			b.WriteString(cursorLine(cur.kind == slotAddOTP, "[+ "+l.T("tui_field_add_otp")+"]"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if m.err != nil {
		b.WriteString(styles.ErrorText.Render(fmt.Sprintf("✗ %v", m.err)))
		b.WriteString("\n\n")
	}
	if m.saving {
		b.WriteString(l.T("tui_saving") + "\n\n")
	}

	// Help line.
	switch m.mode {
	case modeView:
		b.WriteString(styles.HelpText.Render(l.T("tui_help_view")))
	case modeEdit:
		curMultiline := cur.kind == slotField && cur.idx < len(m.defs) && m.defs[cur.idx].multiline
		if m.typeSelectable() && cur.kind == slotType {
			b.WriteString(styles.HelpText.Render(l.T("tui_help_form_type")))
		} else if curMultiline {
			b.WriteString(styles.HelpText.Render(l.T("tui_help_form_multiline")))
		} else {
			b.WriteString(styles.HelpText.Render(l.T("tui_help_form_full")))
		}
	}
	return b.String()
}

func cursorLine(focused bool, text string) string {
	if focused {
		return styles.InputLabel.Render("▸ " + text)
	}
	return "  " + text
}

func labelLine(focused bool, label string) string {
	if focused {
		return styles.InputLabel.Render("▸ " + label + ":")
	}
	return "  " + styles.InputLabel.Render(label+":")
}

// wrapText разбивает строку на подстроки шириной не более width рун, по границам слов
// где возможно. Пустая строка возвращает один пустой элемент (сохраняет пустую строку в выводе).
func wrapText(s string, width int) []string {
	if s == "" {
		return []string{""}
	}
	words := strings.Split(s, " ")
	var lines []string
	cur := ""
	for _, w := range words {
		if cur == "" {
			cur = w
		} else if len([]rune(cur))+1+len([]rune(w)) <= width {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
		}
		// Слово само длиннее width — режем по width рун.
		for len([]rune(cur)) > width {
			r := []rune(cur)
			lines = append(lines, string(r[:width]))
			cur = string(r[width:])
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

func viewFieldLine(focused bool, label, value string) string {
	mark := "  "
	if focused {
		mark = "▸ "
	}
	return fmt.Sprintf("%s%s: %s", mark, styles.InputLabel.Render(label), value)
}

// loadEditData загружает все тиры секрета для заполнения формы.
func loadEditData(ctx context.Context, container *app.Container, vaultID, secretID string, t domain.SecretType, _ int64) (formData, error) {
	fd := formData{fields: map[string]string{}}
	switch t {
	case domain.SecretTypeText:
		d, err := container.Secret.GetTextDetail(ctx, vaultID, secretID)
		if err != nil {
			return fd, err
		}
		fd.fields["title"] = d.Row.Title
		fd.fields["body"] = d.Payload.Body
		fd.fields["note"] = d.Index.Note
		fd.tags = strings.Join(d.Row.Tags, ", ")
		fd.custom = d.Index.CustomFields
		fd.otpCodes = d.Payload.OTPCodes
	case domain.SecretTypeBankCard:
		d, err := container.Secret.GetBankCardDetail(ctx, vaultID, secretID)
		if err != nil {
			return fd, err
		}
		fd.fields["title"] = d.Row.Title
		fd.fields["bank"] = d.Index.Bank
		fd.fields["cardholder"] = d.Index.Cardholder
		fd.fields["brand"] = d.Index.Brand
		fd.fields["expiry"] = d.Index.Expiry
		fd.fields["pan"] = d.Payload.PAN
		fd.fields["cvv"] = d.Payload.CVV
		fd.fields["pin"] = d.Payload.PIN
		fd.fields["note"] = d.Index.Note
		fd.tags = strings.Join(d.Row.Tags, ", ")
		fd.custom = d.Index.CustomFields
		fd.otpCodes = d.Payload.OTPCodes
	case domain.SecretTypeTOTP:
		d, err := container.Secret.GetTOTPDetail(ctx, vaultID, secretID)
		if err != nil {
			return fd, err
		}
		fd.fields["title"] = d.Row.Title
		fd.fields["issuer"] = d.Row.Issuer
		fd.fields["account"] = d.Index.Account
		fd.fields["secret"] = d.Payload.Secret
		fd.fields["note"] = d.Index.Note
		fd.tags = strings.Join(d.Row.Tags, ", ")
		fd.custom = d.Index.CustomFields
		fd.otpCodes = d.Payload.OTPCodes
	case domain.SecretTypeBinary:
		d, err := container.Secret.GetBinaryDetail(ctx, vaultID, secretID)
		if err != nil {
			return fd, err
		}
		fd.fields["title"] = d.Row.Title
		fd.fields["filepath"] = d.Row.Filename // показываем оригинальное имя файла
		fd.fields["note"] = d.Index.Note
		fd.tags = strings.Join(d.Row.Tags, ", ")
		fd.custom = d.Index.CustomFields
		fd.otpCodes = d.Payload.OTPCodes
	default:
		d, err := container.Secret.GetDetail(ctx, vaultID, secretID)
		if err != nil {
			return fd, err
		}
		fd.fields["title"] = d.Row.Title
		fd.fields["uri"] = d.Row.URI
		fd.fields["username"] = d.Row.Username
		fd.fields["password"] = d.Payload.Password
		fd.fields["note"] = d.Index.Note
		fd.tags = strings.Join(d.Row.Tags, ", ")
		fd.custom = d.Index.CustomFields
		fd.otpCodes = d.Payload.OTPCodes
	}
	return fd, nil
}

// formData — данные для наполнения формы при редактировании.
type formData struct {
	fields   map[string]string
	tags     string
	custom   []secretcontent.KeyValue
	otpCodes []secretcontent.OTPCode
}

// --- Валидаторы и маски ввода для полей карты ---

// validateDigits разрешает только цифры.
func validateDigits(s string) error {
	for _, r := range s {
		if r < '0' || r > '9' {
			return fmt.Errorf("digits only")
		}
	}
	return nil
}

// validateExpiry разрешает цифры и один `/` (MM/YY).
func validateExpiry(s string) error {
	slashCount := 0
	for _, r := range s {
		if r == '/' {
			slashCount++
			if slashCount > 1 {
				return fmt.Errorf("invalid")
			}
			continue
		}
		if r < '0' || r > '9' {
			return fmt.Errorf("digits and / only")
		}
	}
	return nil
}

// validatePAN разрешает цифры и пробелы (1234 5678 9012 3456).
func validatePAN(s string) error {
	for _, r := range s {
		if r != ' ' && (r < '0' || r > '9') {
			return fmt.Errorf("digits and spaces only")
		}
	}
	return nil
}

// formatExpiry автоматически вставляет `/` после 2-й цифры: "12" → "12/", "1225" → "12/25".
func formatExpiry(s string) string {
	// Извлекаем только цифры.
	digits := onlyDigits(s)
	if len(digits) > 4 {
		digits = digits[:4]
	}
	if len(digits) <= 2 {
		return digits
	}
	return digits[:2] + "/" + digits[2:]
}

// formatPAN автоматически вставляет пробелы каждые 4 цифры: "1234567890123456" → "1234 5678 9012 3456".
func formatPAN(s string) string {
	digits := onlyDigits(s)
	if len(digits) > 16 {
		digits = digits[:16]
	}
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// onlyDigits извлекает только цифры из строки.
func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// luhnCheckDigit вычисляет контрольную цифру по алгоритму Луна.
func luhnCheckDigit(number string) int {
	sum := 0
	for i := len(number) - 1; i >= 0; i-- {
		d := int(number[i] - '0')
		if (len(number)-i)%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return (10 - sum%10) % 10
}

// luhnValid проверяет номер карты по алгоритму Луна (без пробелов).
func luhnValid(pan string) bool {
	if len(pan) < 12 || len(pan) > 19 {
		return false
	}
	// Проверяем: контрольная цифра (последняя) совпадает с вычисленной для остальных.
	payload := pan[:len(pan)-1]
	check := int(pan[len(pan)-1] - '0')
	return luhnCheckDigit(payload) == check
}

// detectCardBrand определяет платёжную систему по BIN (первым цифрам номера карты).
func detectCardBrand(pan string) string {
	if len(pan) < 4 {
		return ""
	}
	prefix4 := pan[:4]
	switch {
	// Visa: начинается с 4
	case pan[0] == '4':
		return "Visa"
	// Mastercard: 51-55
	case pan[0] == '5' && pan[1] >= '1' && pan[1] <= '5':
		return "Mastercard"
	// Мир: 2200-2204 (проверяем ДО Mastercard 2221-2720, иначе перехватывается)
	case prefix4 >= "2200" && prefix4 <= "2204":
		return "Mir"
	// Mastercard: 2221-2720
	case prefix4 >= "2221" && prefix4 <= "2720":
		return "Mastercard"
	// AmEx: 34 или 37
	case pan[0] == '3' && (pan[1] == '4' || pan[1] == '7'):
		return "AmEx"
	// JCB: 3528-3589
	case prefix4 >= "3528" && prefix4 <= "3589":
		return "JCB"
	// UnionPay: 62
	case pan[0] == '6' && pan[1] == '2':
		return "UnionPay"
	}
	return ""
}

// --- File Picker ---

// handleFilePickerKey обрабатывает клавиши в режиме filepicker.
func (m secretFormModel) handleFilePickerKey(msg tea.KeyMsg) (secretFormModel, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.mode = modeEdit
		m = m.applyFocus()
		return m, nil
	}
	return m.updateFilePicker(msg)
}

// updateFilePicker делегирует в bubbles/filepicker и проверяет выбор файла.
func (m secretFormModel) updateFilePicker(msg tea.Msg) (secretFormModel, tea.Cmd) {
	var cmd tea.Cmd
	m.filePicker, cmd = m.filePicker.Update(msg)

	// Проверяем, выбрал ли пользователь файл.
	if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
		m = m.setValue("filepath", path)
		// Если title пуст — подставляем имя файла.
		if m.value("title") == "" {
			m = m.setValue("title", filepath.Base(path))
		}
		m.dirty = true
		m.mode = modeEdit
		m = m.applyFocus()
		return m, nil
	}
	return m, cmd
}

// handleDownloadPickerKey обрабатывает клавиши в режиме выбора папки назначения для скачивания.
func (m secretFormModel) handleDownloadPickerKey(msg tea.KeyMsg) (secretFormModel, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.mode = modeView
		m = m.applyFocus()
		return m, nil
	}
	return m.updateDownloadPicker(msg)
}

// updateDownloadPicker делегирует в bubbles/filepicker (DirAllowed) и запускает скачивание,
// когда пользователь выбрал директорию.
func (m secretFormModel) updateDownloadPicker(msg tea.Msg) (secretFormModel, tea.Cmd) {
	var cmd tea.Cmd
	m.filePicker, cmd = m.filePicker.Update(msg)

	if didSelect, dir := m.filePicker.DidSelectFile(msg); didSelect {
		m.mode = modeView
		m = m.applyFocus()
		return m, m.downloadBinary(dir)
	}
	return m, cmd
}

// resolveFilePath обрабатывает путь к файлу: если начинается с file:// URI — декодирует,
// если это просто имя файла (без /) — ищет в типичных директориях.
func resolveFilePath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "file://") {
		u, err := url.Parse(raw)
		if err == nil && u.Path != "" {
			return u.Path
		}
	}
	// Расширяем ~ в начале пути.
	if strings.HasPrefix(raw, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, raw[2:])
		}
	}
	// Если абсолютный или относительный путь с / — используем как есть.
	if strings.Contains(raw, "/") || strings.Contains(raw, string(os.PathSeparator)) {
		return raw
	}
	// Просто имя файла — ищем в типичных местах.
	home, _ := os.UserHomeDir()
	searchDirs := []string{
		".",
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Documents"),
		home,
	}
	for _, dir := range searchDirs {
		candidate := filepath.Join(dir, raw)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// Не нашли — возвращаем как есть (ошибка будет при открытии).
	return raw
}

// downloadBinary скачивает binary-секрет в <dir>/<filename> — dir выбирается пользователем
// через flow modeDownloadPicker (Ctrl+... / [D]), а не подставляется тихо (~/Downloads).
func (m secretFormModel) downloadBinary(dir string) tea.Cmd {
	container := m.container
	ctx := m.ctx
	vaultID := m.vaultID
	secretID := m.secretID
	filename := m.value("filepath") // содержит оригинальное имя файла
	if filename == "" {
		filename = "download"
	}

	return func() tea.Msg {
		outPath := filepath.Join(dir, filepath.Base(filename))

		f, err := os.Create(outPath)
		if err != nil {
			return loginErrMsg{err: fmt.Errorf("create file: %w", err)}
		}
		defer func() { _ = f.Close() }()

		if err := container.Secret.DownloadBinary(ctx, vaultID, secretID, f); err != nil {
			_ = os.Remove(outPath)
			return loginErrMsg{err: fmt.Errorf("download: %w", err)}
		}
		return toastMsg{text: fmt.Sprintf(container.Localizer.T("tui_toast_downloaded"), outPath)}
	}
}

// findDuplicateCustomKey возвращает первый дублирующийся ключ custom field (пустые пропускает).
func findDuplicateCustomKey(custom []kvPair) string {
	seen := make(map[string]bool, len(custom))
	for _, kv := range custom {
		k := strings.TrimSpace(kv.key.Value())
		if k == "" {
			continue
		}
		if seen[k] {
			return k
		}
		seen[k] = true
	}
	return ""
}

// findDuplicateOTPCode возвращает первый дублирующийся OTP recovery code (пустые пропускает).
func findDuplicateOTPCode(otps []otpItem) string {
	seen := make(map[string]bool, len(otps))
	for _, o := range otps {
		code := strings.TrimSpace(o.code.Value())
		if code == "" {
			continue
		}
		if seen[code] {
			return code
		}
		seen[code] = true
	}
	return ""
}
