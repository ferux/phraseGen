package api

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ferux/phraseGen/internal/markov"
	"github.com/ferux/phraseGen/internal/pgcontext"

	"github.com/rs/zerolog"
	uuid "github.com/satori/go.uuid"
)

const (
	defaultMaxWordCount = 45
)

// Server implements APIServer gRPC
type Server struct {
	chain        *markov.Chain
	maxWordCount int
}

// NewServer creates a new instance of Server struct with adjusted logrus.
func NewServer(c *markov.Chain, loglevel string) *Server {
	zl := zerolog.New(zerolog.ConsoleWriter{
		Out: os.Stdout,
		FormatTimestamp: func(i interface{}) (out string) {
			out = strconv.Itoa(time.Now().Nanosecond())

			return out
		},
	}).With().Timestamp().Logger()

	zl.Trace().Msg("todo")

	return &Server{
		chain:        c,
		maxWordCount: defaultMaxWordCount,
	}
}

// GetMessage gets request from client, generates new message from Markov Chain.
func (s *Server) GetMessage(ctx context.Context, q *Query) (msg *Message, err error) {
	logger := pgcontext.Zerolog(ctx)

	logger.Debug().Interface("query", q.String()).Msg("proceeding message")

	text, err := s.generateNewSentence(ctx)
	if err != nil {
		return nil, err
	}

	logger.Trace().Msg("creating new message")

	msg = &Message{
		Id:   s.newUUID(),
		Ts:   s.now(),
		Text: text,
	}

	logger.Trace().Msg("returning message")

	return msg, nil
}

// AskStatus returns current status of the application
func (s *Server) AskStatus(ctx context.Context, q *Query) (*Status, error) {
	st := &Status{
		Ts:     s.now(),
		Status: "ok",
	}

	return st, nil
}

func (s *Server) newUUID() string {
	return uuid.NewV4().String()
}

// TODO: convert to state machine.
func (s *Server) generateNewSentence(ctx context.Context) (sentence string, err error) {
	const startCommand = "*START*"

	logger := pgcontext.Zerolog(ctx)

	logger.Trace().Msg("getting new message")

	words := make([]string, 0, s.maxWordCount)
	prev := startCommand

	var count int
	var word string

	var cell markov.Cell
	for {
		logger.Trace().Int("count", count).Str("prev", prev).Msg("getting next word")

		cell, err = s.chain.GetNextWord(prev)
		if err != nil {
			return "", fmt.Errorf("getting next word: %w", err)
		}

		word = cell.GetWord()

		logger.Trace().Str("word", word).Int("text_len", len(words)).Msg("appending new word")

		words = append(words, word)

		if cell.GetType() == markov.End {
			logger.Debug().Int("text_len", len(words)).Msg("message completed")

			break
		}

	}

	logger.Trace().Msg("appending all words together")

	sentence = strings.Join(words[:len(words)-1], " ")
	sentence = strings.Replace(sentence, "*END*", ".", -1)
	sentence = sentence + "."

	logger.Trace().Str("sentence", sentence).Msg("prepared")

	return sentence, nil
}

func (s *Server) now() *Timestamp {
	return &Timestamp{
		Seconds: int64(time.Now().Second()),
		Nanos:   int32(time.Now().Nanosecond()),
	}
}
