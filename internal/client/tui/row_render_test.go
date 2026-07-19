package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func rowRenderLocalizer(t *testing.T) localizerT {
	t.Helper()
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	return c.Localizer
}

func TestColumnsForType_LoginPassword(t *testing.T) {
	cols := columnsForType(domain.SecretTypeLoginPassword, rowRenderLocalizer(t))
	require.Len(t, cols, 4)
}

// computeColumnWidths: узкий терминал — все колонки получают ровно minWidth, лишнего места
// для распределения нет.
func TestComputeColumnWidths_NarrowTerminal_UsesMinWidth(t *testing.T) {
	cols := []tableColumn{
		flexCol("A", 10, 30, 1),
		flexCol("B", 10, 30, 1),
	}
	widths := computeColumnWidths(cols, 20) // ровно сумма minWidth, без остатка
	assert.Equal(t, []int{10, 10}, widths)
}

// На широком терминале свободное место распределяется пропорционально весу.
func TestComputeColumnWidths_WideTerminal_DistributesByWeight(t *testing.T) {
	cols := []tableColumn{
		flexCol("A", 10, 100, 2), // вдвое больший вес
		flexCol("B", 10, 100, 1),
	}
	widths := computeColumnWidths(cols, 40) // +20 сверху minWidth (10+10)
	// 20 делится 2:1 → A +13, B +6, остаток от округления (1) уходит последней flex-колонке (B),
	// чтобы не оставалось неиспользуемого пространства справа от таблицы.
	assert.Equal(t, 10+13, widths[0])
	assert.Equal(t, 10+6+1, widths[1])
	assert.Equal(t, 40, widths[0]+widths[1], "весь availWidth должен быть использован без остатка")
}

// Фиксированная колонка (weight=0) не растёт даже при избытке места.
func TestComputeColumnWidths_FixedColumnDoesNotGrow(t *testing.T) {
	cols := []tableColumn{
		fixedCol("Fixed", 7),
		flexCol("Flex", 10, 100, 1),
	}
	widths := computeColumnWidths(cols, 50)
	assert.Equal(t, 7, widths[0], "фиксированная колонка не должна расти")
	assert.Greater(t, widths[1], 10, "гибкая колонка должна забрать всё свободное место")
}

// Колонка, упёршаяся в maxWidth, перестаёт расти, а излишек перераспределяется оставшимся.
func TestComputeColumnWidths_RespectsMaxWidth(t *testing.T) {
	cols := []tableColumn{
		flexCol("Small", 5, 8, 1), // maxWidth=8, быстро упирается в потолок
		flexCol("Big", 5, 100, 1),
	}
	widths := computeColumnWidths(cols, 50)
	assert.Equal(t, 8, widths[0], "должна остановиться на maxWidth")
	assert.LessOrEqual(t, widths[0]+widths[1], 50)
	assert.Greater(t, widths[1], 8, "оставшееся место должно уйти во вторую колонку")
}

// Регрессия: когда ВСЕ гибкие колонки упираются в maxWidth раньше, чем закончилось доступное
// пространство (типично для наборов с малым числом широких колонок — TOTP/Файлы/вкладка «Все»
// на широком терминале), остаток не должен пропадать пустым хвостом справа от таблицы —
// он уходит последней гибкой колонке.
func TestComputeColumnWidths_AllColumnsHitMaxWidth_RemainderGoesToLastFlex(t *testing.T) {
	cols := []tableColumn{
		flexCol("A", 10, 20, 1),
		flexCol("B", 10, 20, 1),
		fixedCol("C", 10),
	}
	widths := computeColumnWidths(cols, 100) // сильно больше суммы maxWidth (20+20+10=50)
	assert.Equal(t, 20, widths[0], "A должна остановиться на maxWidth")
	assert.Equal(t, 10, widths[2], "фиксированная колонка не растёт")
	total := widths[0] + widths[1] + widths[2]
	assert.Equal(t, 100, total, "весь availWidth должен быть распределён, без пустого хвоста")
}

// Терминал уже суммы minWidth — возвращаем minWidth без паники/отрицательных значений.
func TestComputeColumnWidths_TerminalNarrowerThanMinSum(t *testing.T) {
	cols := []tableColumn{
		flexCol("A", 10, 30, 1),
		flexCol("B", 10, 30, 1),
	}
	widths := computeColumnWidths(cols, 5) // меньше суммы minWidth (20)
	assert.Equal(t, []int{10, 10}, widths)
}

func TestColumnsForType_BankCard(t *testing.T) {
	cols := columnsForType(domain.SecretTypeBankCard, rowRenderLocalizer(t))
	require.Len(t, cols, 7)
}

func TestColumnsForType_TOTP(t *testing.T) {
	cols := columnsForType(domain.SecretTypeTOTP, rowRenderLocalizer(t))
	require.Len(t, cols, 3)
}

func TestColumnsForType_Text(t *testing.T) {
	cols := columnsForType(domain.SecretTypeText, rowRenderLocalizer(t))
	require.Len(t, cols, 2)
}

func TestColumnsForType_Binary(t *testing.T) {
	cols := columnsForType(domain.SecretTypeBinary, rowRenderLocalizer(t))
	require.Len(t, cols, 4)
}

func TestColumnsForType_Default(t *testing.T) {
	cols := columnsForType(0, rowRenderLocalizer(t))
	require.Len(t, cols, 4)
}

func TestRowCells_LoginPassword(t *testing.T) {
	row := secret.SummaryRow{Title: "GitHub", Subtitle: "alice", URI: "github.com"}
	cells := rowCells(domain.SecretTypeLoginPassword, row, "", false, rowRenderLocalizer(t))
	assert.Equal(t, []string{"GitHub", "alice", "github.com", "••••••••"}, cells)
}

func TestRowCells_LoginPassword_Revealed(t *testing.T) {
	row := secret.SummaryRow{Title: "GitHub"}
	cells := rowCells(domain.SecretTypeLoginPassword, row, "hunter2", true, rowRenderLocalizer(t))
	assert.Equal(t, "hunter2", cells[3])
}

func TestRowCells_LoginPassword_RevealedActiveEmpty(t *testing.T) {
	row := secret.SummaryRow{Title: "GitHub"}
	cells := rowCells(domain.SecretTypeLoginPassword, row, "", true, rowRenderLocalizer(t))
	assert.Equal(t, "", cells[3])
}

func TestRowCells_BankCard(t *testing.T) {
	row := secret.SummaryRow{Title: "My Card", Subtitle: "•• 1234", Expiry: "12/29", Cardholder: "Alice", Bank: "Chase", Brand: "Visa"}
	cells := rowCells(domain.SecretTypeBankCard, row, "", false, rowRenderLocalizer(t))
	assert.Equal(t, "My Card", cells[0])
	assert.Equal(t, "•••• •••• •••• 1234", cells[1])
	assert.Equal(t, "12/29", cells[2])
}

func TestRowCells_BankCard_RevealedShowsFullPAN(t *testing.T) {
	row := secret.SummaryRow{Title: "My Card", Subtitle: "•• 1234"}
	cells := rowCells(domain.SecretTypeBankCard, row, "4111111111111111", true, rowRenderLocalizer(t))
	assert.Equal(t, "4111111111111111", cells[1])
}

func TestRowCells_BankCard_HasRevealActionColumn(t *testing.T) {
	row := secret.SummaryRow{Title: "My Card", Subtitle: "•• 1234"}
	cells := rowCells(domain.SecretTypeBankCard, row, "", false, rowRenderLocalizer(t))
	require.Len(t, cells, 7)
	assert.Equal(t, "👁 Show [R]", cells[6])
}

func TestRowCells_TOTP(t *testing.T) {
	row := secret.SummaryRow{Title: "GitHub", Subtitle: "Amazon"}
	cells := rowCells(domain.SecretTypeTOTP, row, "123456", true, rowRenderLocalizer(t))
	assert.Equal(t, []string{"GitHub", "Amazon", "123456"}, cells)
}

func TestRowCells_Text(t *testing.T) {
	row := secret.SummaryRow{Title: "Note"}
	cells := rowCells(domain.SecretTypeText, row, "preview text", true, rowRenderLocalizer(t))
	assert.Equal(t, []string{"Note", "preview text"}, cells)
}

func TestRowCells_Binary(t *testing.T) {
	row := secret.SummaryRow{Title: "Doc", Subtitle: "doc.pdf", Size: 2048}
	cells := rowCells(domain.SecretTypeBinary, row, "", false, rowRenderLocalizer(t))
	assert.Equal(t, []string{"Doc", "doc.pdf", "2 KB", "⬇ Download [D]"}, cells)
}

func TestRowCells_Default(t *testing.T) {
	row := secret.SummaryRow{Title: "GitHub", Subtitle: "alice", Type: int32(domain.SecretTypeLoginPassword)}
	cells := rowCells(0, row, "", false, rowRenderLocalizer(t))
	assert.Equal(t, "🔑", cells[0])
	assert.Equal(t, "GitHub", cells[1])
	assert.Equal(t, "alice", cells[2], "не bank_card — Info = Subtitle как прежде")
}

// На вкладке «Все» карточка должна показывать в Info маскированный номер (последние 4 цифры),
// а в Value — действие «Показать [R]» (номер расшифровывается только по явному запросу).
func TestRowCells_Default_BankCard_ShowsMaskedNumberAndRevealAction(t *testing.T) {
	row := secret.SummaryRow{
		Title: "My Card", Subtitle: "•• 1234", Type: int32(domain.SecretTypeBankCard),
		Bank: "Sber", Cardholder: "IVAN IVANOV", Expiry: "12/34",
	}
	cells := rowCells(0, row, "", false, rowRenderLocalizer(t))
	assert.Equal(t, "💳", cells[0])
	assert.Equal(t, "My Card", cells[1])
	assert.Equal(t, "•••• •••• •••• 1234", cells[2])
	assert.Equal(t, "👁 Show [R]", cells[3])
}

func TestInfoForRow_BankCard_ShowsMaskedNumber(t *testing.T) {
	row := secret.SummaryRow{Subtitle: "•• 1234", Type: int32(domain.SecretTypeBankCard), Bank: "Sber"}
	assert.Equal(t, "•••• •••• •••• 1234", infoForRow(row))
}

func TestHasPayloadColumn(t *testing.T) {
	assert.True(t, hasPayloadColumn(domain.SecretTypeLoginPassword))
	assert.False(t, hasPayloadColumn(domain.SecretTypeBinary))
}

func TestJoinTagsRowRender(t *testing.T) {
	assert.Equal(t, "", joinTags(nil))
	assert.Equal(t, "a", joinTags([]string{"a"}))
	assert.Equal(t, "a, b", joinTags([]string{"a", "b"}))
}

func TestFormatFileSize(t *testing.T) {
	assert.Equal(t, "—", formatFileSize(0))
	assert.Equal(t, "—", formatFileSize(-1))
	assert.Equal(t, "500 B", formatFileSize(500))
	assert.Equal(t, "2 KB", formatFileSize(2048))
	assert.Equal(t, "1.0 MB", formatFileSize(1024*1024))
	assert.Equal(t, "1.0 GB", formatFileSize(1024*1024*1024))
}

func TestMaskCard(t *testing.T) {
	assert.Equal(t, "•••• •••• •••• ••••", maskCard(""))
	assert.Equal(t, "•••• •••• •••• 1234", maskCard("•• 1234"))
	assert.Equal(t, "•••• •••• •••• 1234", maskCard("1234"))
}

func TestTypeShortLabel(t *testing.T) {
	l := rowRenderLocalizer(t)
	assert.Equal(t, "🔑", typeShortLabel(domain.SecretTypeLoginPassword, l))
	assert.Equal(t, "💳", typeShortLabel(domain.SecretTypeBankCard, l))
	assert.Equal(t, "🔢", typeShortLabel(domain.SecretTypeTOTP, l))
	assert.Equal(t, "📝", typeShortLabel(domain.SecretTypeText, l))
	assert.Equal(t, "📁", typeShortLabel(domain.SecretTypeBinary, l))
	assert.Equal(t, "?", typeShortLabel(0, l))
}
