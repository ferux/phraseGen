package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ferux/phraseGen/markov"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// ErrNotReady in case you've made struct instead of using function
var (
	ErrNotReadyResponse = errors.New("internal error")
	ErrNotReady         = errors.New("use NewServer instead of creating &Server{} struct")
)

// Server implements APIServer gRPC
type Server struct {
	C *markov.Chain
	l *logrus.Entry

	ready bool
}

// NewServer creates a new instance of Server struct with adjusted logrus.
func NewServer(c *markov.Chain, loglevel logrus.Level) *Server {
	l := logrus.New()
	l.SetLevel(loglevel)
	le := l.WithField("pkg", "api")
	return &Server{
		C:     c,
		l:     le,
		ready: true,
	}
}

// GetMessage gets request from client, generates new message from Markov Chain.
func (s *Server) GetMessage(ctx context.Context, q *Query) (*Message, error) {
	if !s.ready {
		log.Print(ErrNotReady)
		return nil, ErrNotReadyResponse
	}
	l := s.l.WithField("fn", "GetMessage")
	l.Info("trying to generate new message")
	txt, err := s.getNewMsg()
	if err != nil {
		l.WithError(err).Error("can't get message")
		return &Message{}, err
	}
	l.Debug("preparing new message")
	m := Message{
		Id:   s.newUUID(),
		Ts:   s.now(),
		Text: txt,
	}
	l.WithField("msg", m).Info("message prepared for sent")
	return &m, nil
}

// AskStatus returns current status of the application
func (s *Server) AskStatus(ctx context.Context, q *Query) (*Status, error) {
	if !s.ready {
		log.Print(ErrNotReady)
		return nil, ErrNotReadyResponse
	}
	l := s.l.WithField("fn", "AskStatus")
	st := Status{
		Ts:     s.now(),
		Status: "ok",
	}

	l.WithField("status", st).Info("sending to client how I feel <3")
	return &st, nil
}

func (s *Server) newUUID() string {
	id, _ := uuid.NewV4()
	return id.String()
}

func (s *Server) getNewMsg() (string, error) {
	l := s.l.WithField("fn", "getNewMsg")
	l.Debug("getting new message")
	txts := make([]string, 0)
	prev := "*START*"
	cnt := 0
	for msg, err := s.C.GetNextWord(prev); ; msg, err = s.C.GetNextWord(prev) {
		if err != nil {
			l.WithError(err).Error("can't get new word")
			return "", err
		}
		cnt++
		l.WithFields(logrus.Fields{
			"cnt":    cnt,
			"maxcnt": 30, //too bad
			"msg":    msg.GetWord(),
		}).Debug("got new word to append")
		t := msg.GetWord()
		tp := msg.GetType()
		txts = append(txts, t)
		prev = t
		if tp == markov.End {
			l.Debug("found end of the sentence")
			break
		}
		if cnt > 30 {
			l.Debug("sentence is to big, breaking")
			break
		}
	}
	l.Debug("joining slice into string")
	txt := strings.Join(txts[:len(txts)-1], " ")
	txt = strings.Replace(txt, "*END*", ".", -1)
	txt = txt + "."
	l.WithField("text", txt).Info("finished")
	return fmt.Sprintf("%s", txt), nil
}

func (s *Server) now() *Timestamp {
	return &Timestamp{
		Seconds: int64(time.Now().Second()),
		Nanos:   int32(time.Now().Nanosecond()),
	}
}
