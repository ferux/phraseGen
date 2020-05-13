// Package markov implements markov chains.
// Improvments: Change []Cell to CellContainer for caching all items there.
package markov

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Error string

func (err Error) Error() string { return string(err) }

const (
	// ErrNotFound reports row was not found
	ErrNotFound Error = "not found"

	ErrUnknownError Error = "unknown error"

	ErrEmptyString Error = "empty string"
)

var (

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
	return &Chain{
		d:            make(map[string][]Cell),
		totalRecords: 0,
		asyncMode:    false,
		maxWorkers:   0,
	}
}

// AddCell adds new cell to dictionary. If there's no records of the core string
// new row will be created.
// TODO: make storage instead of map with d.
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
func (c *Chain) GetCells(core string) (cells []Cell, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var ok bool
	cells, ok = c.d[core]
	if !ok {
		return nil, ErrNotFound
	}

	return cells, nil
}

// GetNextWord for generating
func (c *Chain) GetNextWord(core string) (cell Cell, err error) {
	rand.Seed(time.Now().UnixNano())

	cells, err := c.GetCells(core)
	if err != nil {
		return Cell{}, err
	}

	var value float64
	pick := rand.Float64() * 100

	for _, c := range cells {
		value += c.chance

		if pick < value {
			return c, nil
		}
	}

	return Cell{}, ErrUnknownError
}

// GetTotalRecords returns total amount of records
func (c *Chain) GetTotalRecords() uint64 {
	return atomic.LoadUint64(&c.totalRecords)
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

	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetIndent("", " ")

	err = enc.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("encoding data: %w", err)
	}

	return buf.Bytes(), nil
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

	c.d = make(map[string][]Cell, len(c.d))
	c.totalRecords = 0
}

// ChainCoreState stores current state of the process.
// TODO: better naming
type ChainCoreState string

const (
	ChainCoreStateStart = "*START*"
	ChainCoreStateEnd   = "*END*"
)

// ParseText parses the text
func (c *Chain) ParseText(s string) error {
	if len(s) == 0 {
		return ErrEmptyString
	}

	s = strings.TrimSpace(s)
	if s[len(s)-1] != '.' {
		s = s + "."
	}
	s = addSpace(s)

	words := strings.Split(s, " ")
	prevCore := ChainCoreStateStart

	for _, w := range words {
		// Handle parsing better, skipping symbols, etc.
		w = strings.TrimSpace(w)
		if len(w) == 0 {
			continue
		}

		// Add some comments to it.
		switch {
		case w == ".":
			c.AddCell(prevCore, NewCell("*END*", 1, End))
			prevCore = ChainCoreStateStart
		case w[len(w)-1] == 46:
			c.AddCell(prevCore, NewCell(w[:len(w)-1], 1, Word))
			c.AddCell(w[:len(w)-1], NewCell("*END*", 1, End))
			prevCore = ChainCoreStateStart
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

// if it will work -- move to upper vars statements.
var reparse = regexp.MustCompile(`[\wа-яА-Я0-9-]+|[.,:;?!]+`)

// ParseTextV2 uses another regexp to parse input string.
func (c *Chain) ParseTextV2(s string) error {
	if len(s) == 0 {
		return errors.New("string is empty")
	}

	s = strings.TrimSpace(s)
	if s[len(s)-1] != '.' {
		s = s + "."
	}

	words := reparse.FindAllString(s, -1)
	prevCore := "*START*"
	for _, w := range words {
		if len(w) == 0 {
			continue
		}
		switch {
		case w == ".":
			// in case we have smth like that: ...
			if prevCore == "*START*" {
				continue
			}

			c.AddCell(prevCore, NewCell("*END*", 1, End))
			prevCore = "*START*"
		// case w[len(w)-1] == 46:
		// 	c.AddCell(prevCore, NewCell(w[:len(w)-1], 1, Word))
		// 	c.AddCell(w[:len(w)-1], NewCell("*END*", 1, End))
		// 	prevCore = "*START*"
		// case w[len(w)-1] > 32 && w[len(w)-1] < 65:
		// 	continue
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
	if maxworkers < 1 {
		maxworkers = runtime.NumCPU()
	}

	c.asyncMode = true
	c.maxWorkers = maxworkers
}

// RunAsync runs workers.
func (c *Chain) RunAsync() (chan<- string, chan<- struct{}, <-chan error) {
	if !c.asyncMode {
		errc := make(chan error, 1)
		errc <- errors.New("asyncMode if false")
		close(errc)
		return nil, nil, errc
	}

	done := make(chan struct{}, 1)
	bus := make(chan string, 100)
	errc := make(chan error, 20)

	go func() {
		for i := 0; i < c.maxWorkers; i++ {
			c.runWorker(i, bus, errc)
		}

		close(done)
		close(bus)
		close(errc)
	}()

	return bus, done, errc
}

func (c *Chain) runWorker(id int, inc <-chan string, errc chan<- error) {
	var err error

	for msg := range inc {
		if err = c.ParseText(msg); err != nil {
			errc <- err
		}
	}
}

// Export dictionary as byte slice.
func (c *Chain) Export(fname string) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer func(closer io.ReadCloser) {
		if errClose := closer.Close(); errClose != nil {
			// TODO: handle error here
			log.Printf("closing file: %v", errClose)
		}
	}(f)
	enc := gob.NewEncoder(f)
	return enc.Encode(&c.d)
}

// TryImport tries to import cahced file with all parsing results.
func (c *Chain) TryImport(fname string) bool {
	f, err := os.Open(fname)
	if err != nil {
		return false
	}
	defer func(closer io.ReadCloser) {
		if err := closer.Close(); err != nil { /* TODO: handle error */
			log.Printf("closing file: %v", err)
		}
	}(f)

	dec := gob.NewDecoder(f)
	if err := dec.Decode(&c.d); err != nil {
		return false
	}

	return true
}

func addSpace(s string) string {
	return addSpaceRegex.ReplaceAllString(s, "$1 $2")
}
