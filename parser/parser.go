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
	// MaxLine could be larger than actual value. E.g. it could be max-int.
	MaxLine int
}

func (r Range) IsEmpty() bool {
	return r.MinLine == 0
}

type ParserFactory func() (Parser, error)

// Receiver is the interface for receiving the results of a parser.
type Receiver interface {
	// Header of the block. It will be shown any pattern found in this block.
	// The buffer should be available until corresponding EndLevel is called.
	StartLevel(buffer []byte, header Range) error

	// Footer of the block. It will be shown any pattern found in this block.
	EndLevel(buffer []byte, footer Range) error

	// Final level block
	FinalBlock(buffer []byte, body Range) error
}

type ReceiverFunc struct {
	StartLevelFunc func(buffer []byte, header Range) error
	EndLevelFunc   func(buffer []byte, footer Range) error
	FinalBlockFunc func(buffer []byte, body Range) error
}

func (rcvr ReceiverFunc) StartLevel(buffer []byte, header Range) error {
	return rcvr.StartLevelFunc(buffer, header)
}

func (rcvr ReceiverFunc) EndLevel(buffer []byte, footer Range) error {
	return rcvr.EndLevelFunc(buffer, footer)
}

func (rcvr ReceiverFunc) FinalBlock(buffer []byte, body Range) error {
	return rcvr.FinalBlockFunc(buffer, body)
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
