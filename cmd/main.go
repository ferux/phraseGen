package main

import (
	"net"
	"google.golang.org/grpc"
	"github.com/ferux/phraseGen/api"
	"bufio"
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
	var c phrasegen.Configuration
	var err error

	err = godotenv.Load()
	if err != nil {
		panic(err)
	}

	fpath = os.Getenv("GO_FILE")
	c.ErrbitHost = os.Getenv("GO_ERRBIT_HOST")
	c.ErrbitID, err = strconv.ParseInt(os.Getenv("GO_ERRBIT_ID"), 10, 64)
	if err != nil {
		panic(err)
	}
	c.ErrbitKey = os.Getenv("GO_ERRBIT_KEY")
	port, err := strconv.ParseUint(os.Getenv("GO_GRPC_PORT"), 10, 16)
	if err != nil {
		panic(err)
	}
	phrasegen.Port = uint16(port)

	phrasegen.Config = c
	
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
	notifier = phrasegen.Notifier

	phrasegen.Logger = logrus.New()
	l = phrasegen.Logger.WithFields(logrus.Fields{
		"pkg": "main",
		"fn":  "main",
	})
	
}

var (
	notifier *gobrake.Notifier
	// l is for logger
	l *logrus.Entry

	// fpath path to dictionary or text
	fpath string

	status AppStatus
)

const (
	asyncFill = true
	fakeRun = true
)

func main() {
	l.WithFields(logrus.Fields{
		"env":      phrasegen.Environment,
		"version":  phrasegen.Version,
		"revision": phrasegen.Revision,
	}).Info("initializated")

	defer notifier.NotifyOnPanic()
	if phrasegen.Environment == "dev" {
		l.Info("app config: ", phrasegen.Config)
	}

	c := markov.NewChain()

	if !fakeRun {
		if asyncFill {
			fillMarkovAsync(c, fpath)
		} else {
			fillMarkov(c, fpath)
		}
	}

	l.Info("recalculating chances")
	c.CalculateCells()
	l.Println("ready to accept messages")
	sc := bufio.NewReader(os.Stdin)
	status = AppStatus{"Ok"}
	go func() {
		if err := initgRPC(c); err != nil {
			l.WithError(err).Fatal("error in gRPC server")
		}
	} ()
	for {
		fmt.Print("Enter any line. For exit, enter end:\n")
		msg, err := sc.ReadString('\n')
		if err != nil {
			l.WithError(err).Error("can't read from stdin")
			break
		}
		if msg == "end\n" {
			break
		}
		msg = strings.Replace(strings.TrimSpace(getNewMsg(c)), "\n", " ", -1)
		l.Println(msg)
	}

	os.Exit(0)
}

func initgRPC(c *markov.Chain) error {
	lw := l.WithField("fn", "initgRPC")
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", phrasegen.Port))
	if err != nil {
		lw.WithError(err).Error("can't start tcp listener")
		return err
	}
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	api.RegisterAPIServer(grpcServer, &api.Server{C: c})
	lw.WithField("Port", phrasegen.Port).Info("running grpc server")
	
	return grpcServer.Serve(l)
}

func fillMarkov(c *markov.Chain, fpath string) {
	if len(fpath) == 0 {
		l.Warn("no files to parse")
		return
	}

	l.Info("starting bashparser")
	bp := utils.NewBashParser(fpath, logrus.InfoLevel)
	msgc, errc := bp.Start()
	go func() {
		l.Info("started listening to errors")
		for err := range errc {
			l.WithError(err).Error("parsing error")
		}
		l.Info("error channel has been closed")
	}()
	l.Info("ranging throught channel")
	for msg := range msgc {
		_ = c.ParseText(msg)
	}
	if bp.GetStatus() == utils.StatusFail {
		l.Fatal("bashparser ended with error")
	}

}

func fillMarkovAsync(c *markov.Chain, fpath string) {
	if len(fpath) == 0 {
		l.Warn("no files to parse")
		return
	}

	l.Info("starting bashparser")
	bp := utils.NewBashParser(fpath, logrus.InfoLevel)
	msgc, errc := bp.Start()
	go func() {
		l.Info("started listening to errors")
		for err := range errc {
			l.WithError(err).Error("parsing error")
		}
		l.Info("finished listening to errors")
	}()
	l.Info("preparing Chain to work in async mode")
	c.SetAsync(0)
	chainMsgc, chainDonec, chainErrc := c.RunAsync()
	go func() {
		l.Info("started listening to chainParseText errors")
		for err := range chainErrc {
			l.WithError(err).Error("chain.ParsingText")
		}
		l.Info("finished listening to chainParseText errors")
	}()
	l.Info("ranging throught channel")
	for msg := range msgc {
		chainMsgc <- msg
	}
	chainDonec <- struct{}{}
	if bp.GetStatus() == utils.StatusFail {
		l.Fatal("bashparser ended with error")
	}
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
