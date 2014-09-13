package sparser

import (
	"errors"
	"io"

	"github.com/daviddengcn/go-villa"
)

var (
	EOF              = errors.New("EOF")
	UnknownExtension = errors.New("Unknown extension")
)

// All inclusive
type Range struct {
	MinOffs int
	MaxOffs int
	MinLine int
	MaxLine int
}

type ParserFactory func() (Parser, error)

// Receiver is the interface for receiving the results of a parser.
type Receiver interface {
	// Header of the block. It will be shown any pattern found in this block.
	// The buffer should be available until corresponding EndLevel is called.
	StartLevel(buffer []byte, header *Range) error
	
	// Footer of the block. It will be shown any pattern found in this block.
	EndLevel(buffer []byte, footer *Range) error
	
	// Final level block
	FinalBlock(buffer []byte, body *Range) error
}

type Parser interface {
	Parse(in io.Reader, rcvr Receiver) error
}

var factories map[string]ParserFactory = make(map[string]ParserFactory)

func Register(ext string, factory ParserFactory) {
	factories[ext] = factory
}

func New(ext string) (Parser, villa.NestedError) {
	factory, ok := factories[ext]
	if !ok {
		return nil, villa.NestErrorf(UnknownExtension, "extension %s", ext)
	}
	p, err := factory()
	return p, villa.NestErrorf(err, "extension %s", ext)
}
