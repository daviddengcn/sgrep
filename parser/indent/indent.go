package indent

import (
	"bufio"
	"errors"
	"io"

	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sgrep/parser"
)

type Parser struct{}

var (
	EOF_UNEXPECTED = errors.New("EOF unexpected")
	InvalidFormat  = errors.New("Invalid format")
)

const (
	ST_INDENT = iota
	ST_CONTENT
)

func (Parser) Parse(in io.Reader, rcvr sparser.Receiver) error {
	lineNumber := 1
	s := bufio.NewScanner(in)
	var indents villa.IntSlice
	indents.Add(-1)
	buffers := make([][]byte, 1)
	for ; s.Scan(); lineNumber++ {
		line := s.Bytes()
		indent := 0
	lineloop:
		for i, b := range line {
			switch b {
			case ' ':
				indent++
			case '\t':
				indent += 8 - indent%8
			default:
				if b == '#' {
					// ignore comments
					break lineloop
				}
				for indent <= indents[len(indents)-1] {
					if err := rcvr.EndLevel(nil, sparser.Range{}); err != nil {
						return err
					}
					indents.Pop()
				}

				indents.Add(indent)
				if len(indents) > len(buffers) {
					buffers = append(buffers, nil)
				}
				buffers[len(indents)-1] = append(buffers[len(indents)-1][:0], line...)
				rg := sparser.Range{
					MinOffs: i,
					MaxOffs: len(line) - 1,
					MinLine: lineNumber,
					MaxLine: lineNumber,
				}
				if err := rcvr.StartLevel(buffers[len(indents)-1], rg); err != nil {
					return err
				}
				break lineloop
			}
		}
	}
	for len(indents) > 1 {
		if err := rcvr.EndLevel(nil, sparser.Range{}); err != nil {
			return err
		}
		indents.Pop()
	}

	return s.Err()
}
