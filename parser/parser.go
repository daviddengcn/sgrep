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
