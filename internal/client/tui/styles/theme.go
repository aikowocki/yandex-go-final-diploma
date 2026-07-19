package styles

import "github.com/charmbracelet/lipgloss"

// Цвета — адаптивные (light/dark terminal).
var (
	Primary      = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	Secondary    = lipgloss.AdaptiveColor{Light: "#3C3C3C", Dark: "#EEEEEE"}
	Subtle       = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#626262"}
	Success      = lipgloss.AdaptiveColor{Light: "#00A86B", Dark: "#73D68B"}
	Warning      = lipgloss.AdaptiveColor{Light: "#FFB347", Dark: "#FFCC00"}
	Danger       = lipgloss.AdaptiveColor{Light: "#FF6961", Dark: "#FF4444"}
	Highlight    = lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}
	Border       = lipgloss.AdaptiveColor{Light: "#DDDADA", Dark: "#3C3C3C"}
	ActiveBorder = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
)

// Стили элементов.
var (
	// Title — заголовок экрана.
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		MarginBottom(1)

	// Subtitle — подзаголовок.
	Subtitle = lipgloss.NewStyle().
			Foreground(Subtle).
			MarginBottom(1)

	// InputLabel — метка ввода.
	InputLabel = lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true)

	// ErrorText — текст ошибки.
	ErrorText = lipgloss.NewStyle().
			Foreground(Danger).
			Bold(true)

	// SuccessText — текст успеха.
	SuccessText = lipgloss.NewStyle().
			Foreground(Success)

	// HelpText — подсказка.
	HelpText = lipgloss.NewStyle().
			Foreground(Subtle).
			Italic(true)

	// TableBorder — стиль рамки таблицы.
	TableBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Border)

	// ActiveTableBorder — активная таблица.
	ActiveTableBorder = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ActiveBorder)

	// TabActive — активный таб. Без нижней границы (border), чтобы ряд табов занимал ровно
	// одну строку: активность обозначается жирным + цветом, а не линией-подчёркиванием.
	TabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Padding(0, 2)

	// TabInactive — неактивный таб.
	TabInactive = lipgloss.NewStyle().
			Foreground(Subtle).
			Padding(0, 2)

	// Toast — уведомление.
	Toast = lipgloss.NewStyle().
		Background(Primary).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2).
		MarginTop(1)

	// CardBox — бокс для карточки конфликта.
	CardBox = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Border).
		Padding(1, 2).
		Width(40)

	// SelectedCard — выделенная карточка.
	SelectedCard = lipgloss.NewStyle().
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(ActiveBorder).
			Padding(1, 2).
			Width(40)
)

// Centered возвращает стиль, центрирующий контент по горизонтали.
func Centered(width int) lipgloss.Style {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center)
}
