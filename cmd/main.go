package main

import (
	"log"
	"errors"
	"github.com/airbrake/gobrake"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/ferux/phraseGen"
	"github.com/ferux/phraseGen/markov"
	"github.com/ferux/phraseGen/utils"
)

var (
	notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		Host: phraseGen.Config.ErrbitHost,
		ProjectId: phraseGen.Config.ErrbitID,
		ProjectKey: phraseGen.Config.ErrbitKey,
		Environment: phraseGen.Environment,
		Revision: phraseGen.Revision,
	})
	// l is for logger
	l = logrus.New().WithFields(logrus.Fields{
		"Version":  phraseGen.Version,
		"Revision": phraseGen.Revision,
	})
)

func main() {
	l.Info("Started")
	gobrake.SetLogger(log.New(os.Stdout, "errbit ", 0))
	notifier.AddFilter(func (n *gobrake.Notice) *gobrake.Notice {
		n.Params = map[string]interface{}{
			"Version": phraseGen.Version,
			"Revision": phraseGen.Revision,
			"Environment": phraseGen.Environment,
		}
		return n
	})
	notifier.Notify(errors.New("oops"), nil)
	l.Info("Sending notify")

	l.WithError(notifier.Close()).Print("Closed")
	os.Exit(0)
	fn := "./bin/bash.json"

	bp := utils.NewBashParser(fn, logrus.InfoLevel)
	msgc, errc := bp.Start()
	go func() {

		l.Info("Started listening to errors")
		for err := range errc {
			l.WithError(err).Error("got error from parser")
		}
	}()

	c := markov.NewChain()
	l.Info("Ranging throught channel")
	for msg := range msgc {
		_ = c.ParseText(msg)
	}
	c.CalculateCells()
	// c.Iterate()

	l.Println("Ready to accept messages")
	sc := bufio.NewReader(os.Stdin)
	for {
		msg, err := sc.ReadString('\n')
		if err != nil {
			l.WithError(err).Error("can't read from stdin")
			break
		}
		if msg == "end\n" {
			break
		}
		msg = getNewMsg(c)
		l.Println(msg)
	}

	os.Exit(0)
}

func getNewMsg(c *markov.Chain) string {
	txts := make([]string, 0)
	prev := "*START*"
	cnt := 0
	for msg, err := c.GetNextWord(prev); ; msg, err = c.GetNextWord(prev) {
		cnt++
		t := msg.GetWord()
		tp := msg.GetType()
		txts = append(txts, t)
		prev = t
		if tp == markov.End {
			break
		}
		if err != nil {
			// fmt.Println(err)
			break
		}
		if cnt > 30 {
			break
		}
		// fmt.Printf("Word: %s\tType: %d\n", t, tp)
	}
	txt := strings.Join(txts[:len(txts)-1], " ")
	txt = strings.Replace(txt, "*END*", ".", -1)
	txt = txt + "."
	return fmt.Sprintf("%s\n", txt)
}
