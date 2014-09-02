package parser

import (
	"errors"
	"io"
)

var (
	EOF = errors.New("EOF")
)

// All inclusive
type Range struct {
	MinOffs int
	MaxOffs int
	MinLine int
	MaxLine int
}

type Receiver interface {
	// for non-final
	StartLevel(buffer []byte, header *Range) error
	EndLevel(buffer []byte, footer *Range) error
	// final block
	FinalBlock(buffer []byte, header, body, footer *Range) error
}

type Parser interface {
	Parse(in io.Reader, rcvr Receiver) error
}
