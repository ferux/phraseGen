package markov

// Cell is a single cell in Chain
type Cell struct {
	word   string
	count  uint64
	ctype  CellType
	chance float64
}

// NewCell creates new cell
func NewCell(w string, c uint64, t CellType) Cell {
	return Cell{w, c, t, 0}
}

// GetWord returns word
func (c *Cell) GetWord() string {
	return c.word
}

// GetType returns cell type
func (c *Cell) GetType() CellType {
	return c.ctype
}

// Valid ensures Cell is properly set
func (c *Cell) Valid() bool {
	if c.word == "" && c.ctype == Word {
		return false
	}
	if c.word == "*END*" && c.ctype != End {
		return false
	}
	if c.word == "*START*" && c.ctype != Start {
		return false
	}
	if c.count < 1 {
		return false
	}

	return true
}

// ApplyChance calculates chance of appearing current cell in chain
func (c *Cell) ApplyChance(total uint64) {
	if total < c.count {
		return
	}
	c.chance = (float64(c.count) / float64(total)) * 100.0
}
