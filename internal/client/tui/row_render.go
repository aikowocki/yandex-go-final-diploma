package tui

import (
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

// tableColumn — описание колонки таблицы для адаптивной раскладки.
//   - minWidth — минимальная ширина (используется, если места мало или колонка фиксированная).
//   - maxWidth — верхний предел роста при распределении лишнего места (0 = без предела).
//   - weight — вес при распределении свободного пространства терминала (0 = фиксированная
//     ширина, не растёт). Колонки с одинаковым весом растут пропорционально.
//
// Итоговые ширины на конкретный рендер вычисляет computeColumnWidths (dashboard_table.go).
type tableColumn struct {
	title    string
	minWidth int
	maxWidth int
	weight   int
}

// fixedCol — колонка с предсказуемым форматом (иконка типа, срок карты, размер файла), которая
// не должна растягиваться при увеличении ширины терминала.
func fixedCol(title string, width int) tableColumn {
	return tableColumn{title: title, minWidth: width, maxWidth: width, weight: 0}
}

// flexCol — колонка, растягивающаяся пропорционально weight при наличии свободного места
// (после того как все minWidth удовлетворены), но не превышающая maxWidth.
func flexCol(title string, minWidth, maxWidth, weight int) tableColumn {
	return tableColumn{title: title, minWidth: minWidth, maxWidth: maxWidth, weight: weight}
}

// columnsFor возвращает набор колонок для типа секрета (или обобщённый для «Все»/type==0).
// Последняя колонка — «раскрываемое» значение (payload), заполняется по фокусу.
func columnsForType(t domain.SecretType, l localizerT) []tableColumn {
	switch t {
	case domain.SecretTypeLoginPassword:
		return []tableColumn{
			flexCol(l.T("tui_col_title"), 14, 40, 2),
			flexCol(l.T("tui_col_username"), 10, 26, 1),
			flexCol(l.T("tui_col_uri"), 12, 36, 1),
			flexCol(l.T("tui_col_password"), 14, 34, 1),
		}
	case domain.SecretTypeBankCard:
		return []tableColumn{
			flexCol(l.T("tui_col_title"), 10, 24, 1),
			fixedCol(l.T("tui_col_card"), 22), // фиксированный формат "•••• •••• •••• 1234"
			fixedCol(l.T("tui_col_expiry"), 7),
			flexCol(l.T("tui_field_cardholder"), 10, 22, 1),
			flexCol(l.T("tui_field_bank"), 8, 18, 1),
			flexCol(l.T("tui_field_brand"), 6, 14, 1),
			// Отдельная колонка «Действие» с кликабельной подписью "Показать [R]" — раскрывает
			// CVV/PIN/полный номер (то же, что шорткат [R]), вместо неявного маскированного
			// значения где-то среди остальных колонок.
			fixedCol(l.T("tui_col_action"), 16),
		}
	case domain.SecretTypeTOTP:
		return []tableColumn{
			flexCol(l.T("tui_col_title"), 14, 36, 2),
			flexCol(l.T("tui_col_issuer"), 10, 28, 1),
			fixedCol(l.T("tui_col_code"), 16), // код + обратный отсчёт — предсказуемая длина
		}
	case domain.SecretTypeText:
		return []tableColumn{
			flexCol(l.T("tui_col_title"), 14, 40, 1),
			flexCol(l.T("tui_col_preview"), 20, 100, 3), // превью — приоритет на лишнее место
		}
	case domain.SecretTypeBinary:
		return []tableColumn{
			flexCol(l.T("tui_col_title"), 14, 36, 2),
			flexCol(l.T("tui_col_filename"), 14, 36, 2),
			fixedCol(l.T("tui_col_size"), 10),
			// У binary нет раскрываемого payload (hasPayloadColumn==false) — вместо этого
			// здесь кликабельный текст, открывающий флоу скачивания (выбор папки).
			fixedCol(l.T("tui_col_action"), 16),
		}
	default: // «Все» — обобщённый вид
		return []tableColumn{
			fixedCol(l.T("tui_col_type"), 4),
			flexCol(l.T("tui_col_title"), 14, 36, 2),
			flexCol(l.T("tui_col_info"), 14, 36, 2),
			flexCol(l.T("tui_col_value"), 14, 32, 1),
		}
	}
}

// rowCells формирует значения ячеек строки под её тип. revealed — раскрытое чувствительное
// значение текущей строки (пусто, если не раскрыто/не под фокусом); masked — маска для скрытого.
// Последняя ячейка (payload) = revealed или masked.
func rowCells(t domain.SecretType, row secret.SummaryRow, revealed string, revealedActive bool, l localizerT) []string {
	masked := "••••••••"
	payload := masked
	if revealedActive {
		if revealed != "" {
			payload = revealed
		} else {
			payload = ""
		}
	}

	switch t {
	case domain.SecretTypeLoginPassword:
		// Subtitle = username, URI из SummaryRow.URI.
		return []string{row.Title, row.Subtitle, row.URI, payload}
	case domain.SecretTypeBankCard:
		// Card: маскированный номер (из last4), раскрывается полный PAN по фокусу.
		card := maskCard(row.Subtitle)
		if revealedActive && revealed != "" {
			card = revealed // полный PAN
		}
		return []string{row.Title, card, row.Expiry, row.Cardholder, row.Bank, row.Brand, l.T("tui_action_reveal")}
	case domain.SecretTypeTOTP:
		return []string{row.Title, row.Subtitle, payload} // Subtitle = issuer, payload = код
	case domain.SecretTypeText:
		return []string{row.Title, payload} // payload = превью тела
	case domain.SecretTypeBinary:
		// Subtitle = filename. Последняя ячейка — не payload (его нет у binary), а кликабельная
		// подпись "Скачать [D]", открывающая флоу скачивания с выбором папки назначения.
		return []string{row.Title, row.Subtitle, formatFileSize(row.Size), l.T("tui_action_download")}
	default:
		// На вкладке «Все» у binary-строк нет раскрываемого payload — вместо маски точек
		// показываем ту же кликабельную подпись «Скачать», что и на вкладке «Файлы».
		if domain.SecretType(row.Type) == domain.SecretTypeBinary {
			return []string{typeShortLabel(domain.SecretType(row.Type), l), row.Title, infoForRow(row), l.T("tui_action_download")}
		}
		// У bank_card тоже нет обычного raw-раскрытия по фокусу на вкладке «Все» — вместо
		// маски точек в Value показываем действие «Показать [R]» (то же, что на вкладке
		// «Карточки»), а маскированный номер переезжает в Info вместо банка/держателя/срока.
		if domain.SecretType(row.Type) == domain.SecretTypeBankCard {
			return []string{typeShortLabel(domain.SecretType(row.Type), l), row.Title, infoForRow(row), l.T("tui_action_reveal")}
		}
		return []string{typeShortLabel(domain.SecretType(row.Type), l), row.Title, infoForRow(row), payload}
	}
}

// infoForRow формирует значение колонки «Info» на обобщённой вкладке «Все». Для банковских
// карт — маскированный номер (последние 4 цифры видны), поскольку Value-колонка на этой
// вкладке теперь занята действием «Показать [R]», а не самим номером. Для остальных типов —
// Subtitle.
func infoForRow(row secret.SummaryRow) string {
	if domain.SecretType(row.Type) != domain.SecretTypeBankCard {
		return row.Subtitle
	}
	return maskCard(row.Subtitle)
}

// computeColumnWidths распределяет доступную ширину терминала между колонками:
//  1. Каждая колонка сначала получает свой minWidth.
//  2. Остаток (availWidth минус сумма minWidth и разделителей) распределяется между
//     колонками с weight > 0 пропорционально их весу, но не превышая maxWidth (излишек
//     от "упёршихся" в maxWidth колонок перераспределяется между оставшимися).
//  3. Если availWidth меньше суммы minWidth (очень узкий терминал) — возвращаются minWidth
//     без изменений; горизонтальная обрезка средствами truncate() неизбежна в этом случае.
//
// gapPerCol — количество разделительных символов между колонками (используется, чтобы честно
// учитывать реальную занимаемую ширину строки, а не только сумму ширин ячеек).
func computeColumnWidths(cols []tableColumn, availWidth int) []int {
	n := len(cols)
	widths := make([]int, n)
	sumMin := 0
	for i, c := range cols {
		widths[i] = c.minWidth
		sumMin += c.minWidth
	}

	remaining := availWidth - sumMin
	if remaining <= 0 {
		return widths
	}

	// Итеративно раздаём remaining пропорционально весу, "замораживая" колонки, упёршиеся в
	// maxWidth, и раздавая их долю оставшимся — до сходимости (максимум n итераций, колонок
	// в таблице немного, так что цена этого цикла пренебрежимо мала).
	frozen := make([]bool, n)
	lastWeighted := -1 // индекс последней колонки с weight>0 — получит остаток, если все замёрзли
	for i, c := range cols {
		if c.weight > 0 {
			lastWeighted = i
		}
	}
	for pass := 0; pass < n; pass++ {
		totalWeight := 0
		for i, c := range cols {
			if !frozen[i] {
				totalWeight += c.weight
			}
		}
		if totalWeight == 0 || remaining <= 0 {
			break
		}
		anyFrozeThisPass := false
		distributed := 0
		for i, c := range cols {
			if frozen[i] || c.weight == 0 {
				continue
			}
			share := remaining * c.weight / totalWeight
			newWidth := widths[i] + share
			if c.maxWidth > 0 && newWidth > c.maxWidth {
				distributed += c.maxWidth - widths[i]
				widths[i] = c.maxWidth
				frozen[i] = true
				anyFrozeThisPass = true
				continue
			}
			widths[i] = newWidth
			distributed += share
		}
		remaining -= distributed
		if !anyFrozeThisPass {
			break
		}
	}
	// Если после распределения остался излишек (все гибкие колонки упёрлись в maxWidth раньше,
	// чем было исчерпано доступное пространство — типично для наборов с малым числом широких
	// flex-колонок, например TOTP/Файлы) — отдаём его последней гибкой колонке сверх её maxWidth,
	// а не оставляем пустым хвостом справа от таблицы. Такой перерасход визуально приемлем
	// (дополнительное место для длинных значений), в отличие от неиспользуемого пространства.
	if remaining > 0 && lastWeighted >= 0 {
		widths[lastWeighted] += remaining
	}
	return widths
}

// hasPayloadColumn сообщает, есть ли у типа раскрываемая payload-колонка (у binary — нет).
func hasPayloadColumn(t domain.SecretType) bool {
	return t != domain.SecretTypeBinary
}

// payloadColumnIndex возвращает индекс колонки с раскрываемым чувствительным значением для
// визуальной обратной связи «✓ copied». У большинства типов это последняя колонка, но у bank_card
// последняя колонка отдельное действие «Показать [R]».
func payloadColumnIndex(t domain.SecretType, numCols int) int {
	if t == domain.SecretTypeBankCard {
		return 1
	}
	return numCols - 1
}

func joinTags(tags []string) string {
	out := ""
	for i, t := range tags {
		if i > 0 {
			out += ", "
		}
		out += t
	}
	return out
}

// formatFileSize форматирует размер файла в человекочитаемый вид (KB/MB/GB).
func formatFileSize(size int64) string {
	if size <= 0 {
		return "—"
	}
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case size >= gb:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(gb))
	case size >= mb:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(mb))
	case size >= kb:
		return fmt.Sprintf("%d KB", size/kb)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// maskCard формирует маскированный номер карты из last4: "•••• •••• •••• 3456".
// Если last4 пустой — возвращает просто маску. Если в subtitle уже есть "•• " — убираем.
func maskCard(subtitle string) string {
	last4 := subtitle
	// SummaryRow.Subtitle для карты = "•• 1234" — извлекаем последние 4 цифры.
	if len(subtitle) >= 4 {
		last4 = subtitle[len(subtitle)-4:]
	}
	if last4 == "" {
		return "•••• •••• •••• ••••"
	}
	return "•••• •••• •••• " + last4
}

// typeShortLabel — иконка типа для колонки «Type» на вкладке «Все» (компактнее текста).
func typeShortLabel(t domain.SecretType, l localizerT) string {
	switch t {
	case domain.SecretTypeLoginPassword:
		return "🔑"
	case domain.SecretTypeBankCard:
		return "💳"
	case domain.SecretTypeTOTP:
		return "🔢"
	case domain.SecretTypeText:
		return "📝"
	case domain.SecretTypeBinary:
		return "📁"
	default:
		return "?"
	}
}
