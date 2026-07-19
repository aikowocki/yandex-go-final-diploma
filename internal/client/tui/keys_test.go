package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestCyrillicShortcuts проверяет, что однобуквенные шорткаты dashboard (n/s/u/l/c и т.д.)
// сверяются корректно, даже если символ приходит как кириллическая руна той же физической
// клавиши QWERTY (эмуляция русской раскладки клавиатуры: терминал отдаёт именно то, что
// печатает активная системная раскладка, поэтому "N" на ЙЦУКЕН приходит как "т").
func TestCyrillicShortcuts(t *testing.T) {
	cases := []struct {
		name     string
		cyrillic rune
		shortcut string
	}{
		{"n (создать) -> т", 'т', "n"},
		{"s (настройки) -> ы", 'ы', "s"},
		{"u (юзер) -> г", 'г', "u"},
		{"l (заблокировать) -> д", 'д', "l"},
		{"c (копировать) -> с", 'с', "c"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.cyrillic}}
			if !isShortcut(msg, tc.shortcut) {
				t.Errorf("normalizeKey(%q) = %q, want shortcut %q to match", tc.cyrillic, normalizeKey(msg), tc.shortcut)
			}
		})
	}
}

// TestNormalizeKeyPassthrough проверяет, что нормализация не трогает уже-ASCII клавиши
// (латинские буквы, цифры, служебные клавиши) — раскладка EN должна работать как раньше.
func TestNormalizeKeyPassthrough(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	if got := normalizeKey(msg); got != "n" {
		t.Errorf("normalizeKey('n') = %q, want %q", got, "n")
	}

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	if got := normalizeKey(enter); got != "enter" {
		t.Errorf("normalizeKey(enter) = %q, want %q", got, "enter")
	}
}

// TestCyrillicDoesNotFalsePositive проверяет, что кириллическая буква, не участвующая
// в шорткатах, не совпадает случайно с другим шорткатом.
func TestCyrillicDoesNotFalsePositive(t *testing.T) {
	// 'п' физически соответствует 'g' на QWERTY — не должна совпасть с "n".
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'п'}}
	if isShortcut(msg, "n") {
		t.Errorf("'п' (-> g) should not match shortcut 'n'")
	}
	if !isShortcut(msg, "g") {
		t.Errorf("'п' should map to 'g'")
	}
}

// TestVaultSwitchBrackets проверяет, что скобки [ ] (переключение vault-табов) распознаются
// как на латинской раскладке, так и на кириллической (х -> [, ъ -> ]).
func TestVaultSwitchBrackets(t *testing.T) {
	cases := []struct {
		name     string
		r        rune
		shortcut string
	}{
		{"latin [", '[', "["},
		{"latin ]", ']', "]"},
		{"cyrillic х -> [", 'х', "["},
		{"cyrillic ъ -> ]", 'ъ', "]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.r}}
			if !isShortcut(msg, tc.shortcut) {
				t.Errorf("normalizeKey(%q) = %q, want %q", tc.r, normalizeKey(msg), tc.shortcut)
			}
		})
	}
}
