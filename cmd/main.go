package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/namsral/flag"

	"github.com/airbrake/gobrake"
	"github.com/sirupsen/logrus"

	"github.com/ferux/phraseGen"
	"github.com/ferux/phrasegen/markov"
	"github.com/ferux/phrasegen/utils"
)

func init() {
	flag.String(flag.DefaultConfigFlagname, "", "Config file")
	c := phrasegen.Configuration{}
	c.ErrbitHost = *flag.String("ERRBIT_HOST", "", "Errbit Host")
	c.ErrbitID = *flag.Int64("ERRBIT_ID", 0, "Errbit Project ID")
	c.ErrbitKey = *flag.String("ERRBIT_KEY", "", "Errbit Project Key")
	phrasegen.Config = c
	phrasegen.Environment = *flag.String("ENV", "Develop", "Environment")
	phrasegen.Logger = logrus.New()

	flag.Parse()

	phrasegen.Notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		Host:        phrasegen.Config.ErrbitHost,
		ProjectId:   phrasegen.Config.ErrbitID,
		ProjectKey:  phrasegen.Config.ErrbitKey,
		Environment: phrasegen.Environment,
		Revision:    phrasegen.Revision,
	})
}

var (
	notifier = phrasegen.Notifier
	// l is for logger
	l = phrasegen.Logger.WithFields(logrus.Fields{
		"Version":  phrasegen.Version,
		"Revision": phrasegen.Revision,
	})
)

func main() {
	l.Info("Started")
	notifier.AddFilter(func(n *gobrake.Notice) *gobrake.Notice {
		n.Params = map[string]interface{}{
			"Version":     phrasegen.Version,
			"Revision":    phrasegen.Revision,
			"Environment": phrasegen.Environment,
		}
		return n
	})

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
