package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/airbrake/gobrake"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"github.com/ferux/phraseGen"
	"github.com/ferux/phraseGen/markov"
	"github.com/ferux/phraseGen/utils"
)

func init() {
	var err error
	err = godotenv.Load()
	if err != nil {
		panic(err)
	}

	if fpath = os.Getenv("GO_FILE"); len(fpath) == 0 {
		fpath = *flag.String("file", "", "Path to file")
	}

	flag.Parse()

	c := phrasegen.Configuration{}
	c.ErrbitHost = os.Getenv("GO_ERRBIT_HOST")
	c.ErrbitID, err = strconv.ParseInt(os.Getenv("GO_ERRBIT_ID"), 10, 64)

	if err != nil {
		panic(err)
	}
	c.ErrbitKey = os.Getenv("GO_ERRBIT_KEY")
	phrasegen.Config = c
	phrasegen.Environment = os.Getenv("GO_ENV")
	phrasegen.Logger = logrus.New()

	phrasegen.Notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		Host:        phrasegen.Config.ErrbitHost,
		ProjectId:   phrasegen.Config.ErrbitID,
		ProjectKey:  phrasegen.Config.ErrbitKey,
		Environment: phrasegen.Environment,
		Revision:    phrasegen.Revision,
	})

	phrasegen.Notifier.AddFilter(func(n *gobrake.Notice) *gobrake.Notice {
		n.Params = map[string]interface{}{
			"version":     phrasegen.Version,
			"revision":    phrasegen.Revision,
			"environment": phrasegen.Environment,
		}
		return n
	})

	l = phrasegen.Logger.WithFields(logrus.Fields{
		"version":  phrasegen.Version,
		"revision": phrasegen.Revision,
		"pkg":      "main",
		"fn":       "main",
	})
	notifier = phrasegen.Notifier
}

var (
	notifier *gobrake.Notifier
	// l is for logger
	l *logrus.Entry

	// fpath path to dictionary or text
	fpath string
)

func main() {
	l.Info("Started")
	defer notifier.NotifyOnPanic()
	if phrasegen.Environment == "dev" {
		l.WithField("Configuration", phrasegen.Config)
	}

	bp := utils.NewBashParser(fpath, logrus.InfoLevel)
	msgc, errc := bp.Start()
	go func() {
		l.Info("Started listening to errors")
		for err := range errc {
			l.WithError(err).Error("parsing error")
		}
		l.Info("Error channel has been closed")
	}()

	c := markov.NewChain()
	l.Info("Ranging throught channel")
	for msg := range msgc {
		_ = c.ParseText(msg)
	}
	l.Info("Recalculating chances")
	c.CalculateCells()
	// c.Iterate()
	if bp.GetStatus() == utils.StatusFail {
		l.Fatal("bashparser ended with error")
	}
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
