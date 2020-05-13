package markov

// CellType for describing type of each cell.
type CellType int

// Enums for CellType
const (
	Start CellType = iota
	Word
	End
)

func (c CellType) String() string {
	switch c {
	case Start:
		return "start"
	case Word:
		return "word"
	case End:
		return "end"
	default:
		return "unknown"
	}
}

// TextFormatter formats text excluding all useless information.
type TextFormatter interface {
	Format(string) string
}
