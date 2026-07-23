package tui

import tea "github.com/charmbracelet/bubbletea"

// ruToEnLayout — соответствие символов ЙЦУКЕН (нижний регистр) их ASCII-эквивалентам на той же
// физической клавише QWERTY. Составлено по стандартной раскладке Windows/macOS "Русская".
var ruToEnLayout = map[rune]rune{
	'й': 'q', 'ц': 'w', 'у': 'e', 'к': 'r', 'е': 't', 'н': 'y', 'г': 'u', 'ш': 'i', 'щ': 'o', 'з': 'p',
	'ф': 'a', 'ы': 's', 'в': 'd', 'а': 'f', 'п': 'g', 'р': 'h', 'о': 'j', 'л': 'k', 'д': 'l',
	'я': 'z', 'ч': 'x', 'с': 'c', 'м': 'v', 'и': 'b', 'т': 'n', 'ь': 'm',
	'б': ',', 'ю': '.', 'ж': ';', 'э': '\'', 'х': '[', 'ъ': ']', 'ё': '`',
}

// normalizeKey возвращает ASCII-эквивалент нажатия для сверки с шорткатами.
// Для клавиш, уже пришедших как ASCII (в т.ч. цифры, Enter, Tab, Ctrl+X), возвращает msg.String()
// без изменений — нормализация касается только кириллических рун.
func normalizeKey(msg tea.KeyMsg) string {
	s := msg.String()
	if len(msg.Runes) == 1 {
		if mapped, ok := ruToEnLayout[msg.Runes[0]]; ok {
			return string(mapped)
		}
	}
	return s
}

// isShortcut сравнивает нормализованную клавишу с ASCII-шорткатом (одна буква, ожидается
// уже в нижнем регистре).
func isShortcut(msg tea.KeyMsg, shortcut string) bool {
	return normalizeKey(msg) == shortcut
}
