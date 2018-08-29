// Package markov implements markov chains.
// Improvments: Change []Cell to CellContainer for caching all items there.
package markov

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	// ErrNotFound reports row was not found
	ErrNotFound = errors.New("not found")
	// l logger for logging package messages
	l = logrus.New().WithField("pkg", "markov")

	// checkSymbols regexp for finding symbols only
	// checkSymbolsRegex = regexp.MustCompile(`^[\W|\D]$`)

	// addSpace regexp searches throught words and marks symbols attached to these words.
	addSpaceRegex = regexp.MustCompile(`(?m)([а-яА-Я\w\-]+)([.,;:!?\(\)\"\'])`)
)

// Chain contains dictionary of parsed text
type Chain struct {
	d            map[string][]Cell
	totalRecords uint64

	// determines if TextProcessing runs in concurrency mode
	asyncMode  bool
	maxWorkers int
	mu         sync.RWMutex
}

// NewChain creates new chain
// nolint
func NewChain() *Chain {
	l.Info("Created new Chain")
	return &Chain{make(map[string][]Cell), 0, false, 0, sync.RWMutex{}}
}

// AddCell adds new cell to dictionary. If there's no  records of the core string
// new row will be created.
func (c *Chain) AddCell(core string, cell Cell) {
	if !cell.Valid() {
		fmt.Println("Cell is not valid. Skipping.")
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalRecords++
	_, ok := c.d[core]
	if !ok {
		c.d[core] = make([]Cell, 1)
		c.d[core][0] = cell
		return
	}
	for i := range c.d[core] {
		if c.d[core][i].word == cell.word {
			c.d[core][i].count++
			return
		}
	}
	c.d[core] = append(c.d[core], cell)
}

// GetCells gets cell slice of core
func (c *Chain) GetCells(core string) ([]Cell, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if ca, ok := c.d[core]; ok {
		return ca, nil
	}
	fmt.Printf("Core: %s\tNot found\n", core)
	return nil, ErrNotFound
}

// GetNextWord for generating
func (c *Chain) GetNextWord(core string) (Cell, error) {
	rand.Seed(time.Now().UnixNano())
	c.mu.RLock()
	cells, err := c.GetCells(core)
	c.mu.RUnlock()
	if err != nil {
		return Cell{}, err
	}
	pick := rand.Float64() * 100
	value := 0.00

	for _, c := range cells {
		value += c.chance
		// fmt.Printf("Rolling dice: %6.2f%%\tChance: %6.2f%%\n", pick, value)
		if pick < value {
			return c, nil
		}
	}
	return Cell{}, errors.New("something went wrong")
}

// GetTotalRecords returns total amount of records
func (c *Chain) GetTotalRecords() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.totalRecords
}

// CalculateCells sets chance of appearance of each cell.
func (c *Chain) CalculateCells() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.d {
		var total uint64
		for _, vc := range c.d[k] {
			total += vc.count
		}
		for ck := range c.d[k] {
			c.d[k][ck].ApplyChance(total)
		}
	}
}

// JSON generates JSON output for dictionary.
func (c *Chain) JSON() ([]byte, error) {
	return json.Marshal(c.d)
}

// Beautify adds ident to json.
func (c *Chain) Beautify(data []byte, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(make([]byte, 0))
	err = json.Indent(buf, data, "", " ")
	return buf.Bytes(), err
}

// Iterate throught dictionary.
func (c *Chain) Iterate() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for k, v := range c.d {
		fmt.Printf("Word: %s [\n", k)
		for _, cell := range v {
			fmt.Printf("\tWord: %20s\tCount: %d\tChance: %6.2f%%\n", cell.word, cell.count, cell.chance)
		}
		fmt.Println("]")
	}
}

// Reset erases all rows from dictionary.
func (c *Chain) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.d = make(map[string][]Cell)
	c.totalRecords = 0
}

// ParseText parses the text
func (c *Chain) ParseText(s string) error {
	if len(s) == 0 {
		return errors.New("string is empty")
	}

	s = strings.TrimSpace(s)
	if s[len(s)-1] != '.' {
		s = s + "."
	}
	s = addSpace(s)

	words := strings.Split(s, " ")
	prevCore := "*START*"
	for _, w := range words {
		w = strings.TrimSpace(w)
		if len(w) == 0 {
			continue
		}
		switch {
		case w == ".":
			c.AddCell(prevCore, NewCell("*END*", 1, End))
			prevCore = "*START*"
		case w[len(w)-1] == 46:
			c.AddCell(prevCore, NewCell(w[:len(w)-1], 1, Word))
			c.AddCell(w[:len(w)-1], NewCell("*END*", 1, End))
			prevCore = "*START*"
		case w[len(w)-1] > 32 && w[len(w)-1] < 65:

			continue
		default:
			wl := strings.ToLower(w)
			cell := NewCell(wl, 1, Word)
			c.AddCell(prevCore, cell)
			prevCore = wl
		}
	}
	return nil
}

// SetAsync turns text processing in concurrency mode. You should also specify number of workers
// which should be more than 1 or will be binded to amount of cpu cores.
func (c *Chain) SetAsync(maxworkers int) {
	lw := l.WithField("fn", "SetAsync")
	if maxworkers < 1 {
		maxworkers = runtime.NumCPU()
	}
	lw.WithField("maxWorkers", maxworkers).Info("Setting up Async Chain")
	c.asyncMode = true
	c.maxWorkers = maxworkers
}

// RunAsync runs workers.
func (c *Chain) RunAsync() (chan<- string, chan<- struct{}, <-chan error) {
	lw := l.WithField("fn", "RunAsync")
	if !c.asyncMode {
		lw.Warn("not in async mode")
		errc := make(chan error, 1)
		errc <- errors.New("asyncMode if false")
		close(errc)
		return nil, nil, errc
	}
	lw.Info("preparing channels")
	donec := make(chan struct{}, 1)
	inc := make(chan string, 100)
	errc := make(chan error, 20)

	go func() {
		for i := 0; i < c.maxWorkers; i++ {
			lw.WithField("id", i).Info("Running worker")
			c.runWorker(i, inc, errc)
		}
		<-donec
		close(donec)
		close(inc)
		close(errc)
	}()
	return inc, donec, errc
}

func (c *Chain) runWorker(id int, inc <-chan string, errc chan<- error) {
	var err error
	lw := l.WithField("WorkerID", id)
	lw.Info("Started")
	defer lw.Info("Finished")

	for msg := range inc {
		if err = c.ParseText(msg); err != nil {
			errc <- err
		}
	}
}

func addSpace(s string) string {
	return addSpaceRegex.ReplaceAllString(s, "$1 $2")
}
