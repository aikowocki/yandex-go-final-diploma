package localstore

// scanner абстрагирует *sql.Row и *sql.Rows для общих scan-хелперов.
type scanner interface {
	Scan(dest ...any) error
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
