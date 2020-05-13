package markov

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// CellV2 should be faster than cell.
type CellV2 map[string]*CellInfo

// CellInfo contains nameless information about cell.
type CellInfo struct {
	Count  int64
	Ctype  CellType
	Chance float64
	mu     sync.Mutex
}

// AddCount increases count value by 1.
func (ci *CellInfo) AddCount() {
	atomic.AddInt64(&ci.Count, 1)
}

// Valid checks if the word and CellInfo are valid.
func (c *CellInfo) Valid(word string) bool {
	if word == "" && c.Ctype == Word {
		return false
	}
	if word == "*END*" && c.Ctype != End {
		return false
	}
	if word == "*START*" && c.Ctype != Start {
		return false
	}
	if c.Count < 1 {
		return false
	}
	return true
}

func extractCellInfo(c Cell) *CellInfo {
	return &CellInfo{
		Count:  int64(c.count),
		Ctype:  c.ctype,
		Chance: c.chance,
	}
}

// AddCell adds new cell to map.
func (c CellV2) AddCell(cell Cell) {
	if !cell.Valid() {
		return
	}
	word := cell.GetWord()
	ci, ok := c[word]
	if !ok {
		c[word] = extractCellInfo(cell)
		return
	}
	ci.AddCount()
}

// ApplyChance calculates chance of appearing current cell in chain
func (c *CellInfo) ApplyChance(total uint64) {
	if total < uint64(c.Count) {
		return
	}

	c.mu.Lock()
	c.Chance = (float64(c.Count) / float64(total)) * 100.0
	c.mu.Unlock()
}

func (c *CellInfo) String() string {
	return fmt.Sprintf("Cell of type %s found %d time(s) with chance %.2f", c.Ctype.String(), c.Count, c.Chance)
}
