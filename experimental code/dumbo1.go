package cleisthenes

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/rs/zerolog"
	"os"
	"time"
)

type Epoch uint64

type Dumbo1 interface {
	HandleContribution(contribution Contribution)
	HandleMessage(msg *pb.Message) error
	OnConsensus() bool
	Close()
}

var (
	log zerolog.Logger
)

func NewLoggerWithHead(head string) zerolog.Logger {
	cw := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.StampMicro}
	cw.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%6s | %s", head, i)
	}
	cw.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s:", i)
	}
	logger := zerolog.New(cw).With().Timestamp().Logger()
	return logger
}

func NewLogger() zerolog.Logger {
	cw := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.StampMicro}
	logger := zerolog.New(cw).With().Timestamp().Logger()
	//zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	return logger
}

func init() {
	log = NewLogger()
}
