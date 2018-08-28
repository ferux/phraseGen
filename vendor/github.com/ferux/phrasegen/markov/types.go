package markov

// CellType for describing type of each cell.
type CellType int

// Enums for CellType
const (
	Start CellType = iota
	Word
	End
)
