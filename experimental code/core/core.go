package core

import (
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/sliceSystem"
	"github.com/rs/zerolog"
)

var (
	log zerolog.Logger
)

func init() {
	log = cleisthenes.NewLoggerWithHead("CORE")
}

func NewSliceSystem(validator cleisthenes.TxValidator) (sliceSystem.SliceSystem, error) {
	return sliceSystem.New(validator)
}
