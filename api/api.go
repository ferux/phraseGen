package api

import (
	"fmt"
	"strings"
	"context"
	"time"
	"github.com/satori/go.uuid"
	"github.com/ferux/phraseGen/markov"
)

/*
type APIServer interface {
	// GetMessage asks Chains to generate new random message and send it
	// to the client.
	GetMessage(context.Context, *Query) (*Message, error)
	// AskStatus asks server about its status.
	AskStatus(context.Context, *Query) (*Status, error)
}
*/

// Server implements APIServer gRPC
type Server struct {
	C *markov.Chain
}

// GetMessage gets request from client, generates new message from Markov Chain.
func (s *Server) GetMessage(ctx context.Context, q *Query) (*Message, error) {
	txt, err := s.getNewMsg()
	if err != nil {
		return &Message{}, err
	}
	
	return &Message{
		Id: s.newUUID(),
		Ts: s.now(),
		Text: txt,
	}, nil
}
// AskStatus returns current status of the application
func (s *Server) AskStatus(ctx context.Context, q *Query) (*Status, error) {
	return &Status{
		Ts: s.now(),
		Status: "ok",
	}, nil
}

func (s *Server) newUUID() string {
	id, _ := uuid.NewV4()
	return id.String()
}

func (s *Server) getNewMsg() (string, error) {
	txts := make([]string, 0)
	prev := "*START*"
	cnt := 0
	for msg, err := s.C.GetNextWord(prev); ; msg, err = s.C.GetNextWord(prev) {
		cnt++
		t := msg.GetWord()
		tp := msg.GetType()
		txts = append(txts, t)
		prev = t
		if tp == markov.End {
			break
		}
		if err != nil {
			return "", err
		}
		if cnt > 30 {
			break
		}
		// fmt.Printf("Word: %s\tType: %d\n", t, tp)
	}
	txt := strings.Join(txts[:len(txts)-1], " ")
	txt = strings.Replace(txt, "*END*", ".", -1)
	txt = txt + "."
	return fmt.Sprintf("%s\n", txt), nil
}

func (s *Server) now() *Timestamp {
	return &Timestamp{
		Seconds: int64(time.Now().Second()),
		Nanos: int32(time.Now().Nanosecond()),
	}
}