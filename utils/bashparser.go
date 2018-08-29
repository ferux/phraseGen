package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// BashParser provide file parsing to bash quotes
type BashParser struct {
	filename string
	status   ParseStatus
	ready    bool
	fast     bool

	bashQuotes []BashStruct

	outc chan string
	errc chan error
	done chan struct{}

	l *logrus.Entry
}

// NewBashParser creates new parser.
func NewBashParser(filename string, loglevel logrus.Level) *BashParser {
	bp := &BashParser{
		filename:   filename,
		bashQuotes: make([]BashStruct, 0),

		fast: true,

		done: make(chan struct{}, 1),

		l: logrus.New().WithFields(logrus.Fields{
			"pkg": "utils",
			"obj": "BashParser",
		}),
	}
	bp.l.Level = loglevel

	return bp
}

// GetChannel returns channel to get strings for parsing
func (b *BashParser) GetChannel() <-chan string {
	return b.outc
}

// Close parser
func (b *BashParser) Close() {
	b.done <- struct{}{}
}

// Start creates channels and runs loop for processing file.
func (b *BashParser) Start() (<-chan string, <-chan error) {

	b.outc = make(chan string, 100)
	b.errc = make(chan error, 100)
	if b.fast {
		go b.fastloop()
	} else {
		go b.loop()
	}
	return b.outc, b.errc
}

// GetStatus returns status of application
func (b *BashParser) GetStatus() ParseStatus {
	return b.status
}

func (b *BashParser) loop() {
	// logging is good
	l := b.l.WithFields(logrus.Fields{
		"fn":   "loop",
		"file": b.filename,
	})
	l.Info("Started loop")

	// open file
	f, err := os.Open(b.filename)
	if err != nil {
		b.status = StatusFail
		l.WithError(err).Error("exiting")
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			l.WithError(err).Error("can't close file")
		}
		l.Info("Finished loop")
		close(b.outc)
		close(b.errc)
		close(b.done)
	}()

	dec := json.NewDecoder(f)
	if _, err := dec.Token(); err != nil {
		b.status = StatusFail
		l.WithError(err).Error("can't extract open token")
		b.errc <- err
		return
	}
	var quote BashStruct
	var rows uint32
	start := time.Now()
	spent := ""
	go func() {
		for range time.Tick(time.Second) {
			spent = time.Since(start).String()
		}
	}()
	for dec.More() {
		if err := dec.Decode(&quote); err != nil {
			b.status = StatusWarn
			l.WithError(err).Error("can't decode row, skipping")
			b.errc <- err
			continue
		}
		b.outc <- quote.Text
		rows++
		fmt.Print("\033[2K\r")
		fmt.Printf("Processing row: %d (%s)", rows, spent)
	}

	if _, err := dec.Token(); err != nil {
		b.status = StatusFail
		l.WithError(err).Error("can't extract close token")
		b.errc <- err
		return
	}
	l.Infof("rows proceeded: %d for %s", rows, time.Since(start).String())
	b.status = StatusOk
}

func (b *BashParser) fastloop() {
	l := b.l.WithFields(logrus.Fields{
		"fn":   "fastloop",
		"file": b.filename,
	})
	l.Info("Started loop")

	defer func() {
		close(b.outc)
		close(b.errc)
		close(b.done)
	}()
	// open file
	f, err := ioutil.ReadFile(b.filename)
	if err != nil {
		b.status = StatusFail
		l.WithError(err).Error("exiting")
		b.errc <- err
		return
	}
	defer func() {
		l.Info("Finished loop")
	}()

	var bqs []BashStruct
	if err := json.Unmarshal(f, &bqs); err != nil {
		b.status = StatusFail
		l.WithError(err).Error("can't unmarshal")
		b.errc <- err
		return
	}
	var rows, rps, secs uint64
	total := uint64(len(bqs))
	start := time.Now()
	t := time.NewTicker(time.Second)
	go func() {
		for range t.C {
			secs++
			fmt.Print("\033[2K\r")
			fmt.Printf("Processing row (%7d/%7d) %d rps (%d avg)\tEstimated: %d seconds", rows, total, rps, rows/secs, (total-rows)/(rows/secs))
			rps = 0
		}
	}()
	for _, bq := range bqs {
		b.outc <- filterBashDialog(bq.Text)
		rows++
		rps++
	}
	t.Stop()
	fmt.Print("\033[2K\r")
	l.WithFields(logrus.Fields{
		"rows":     rows,
		"avg(rps)": rows / secs,
	}).Infof("Task done for %.3f second(s)", time.Since(start).Seconds())
	b.status = StatusOk
}

