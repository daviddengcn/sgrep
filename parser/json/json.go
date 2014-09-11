package json

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"text/scanner"
	"unicode"

	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sgrep/parser"
)

type Parser struct{}

var (
	EOF_UNEXPECTED = errors.New("EOF unexpected")
	InvalidFormat  = errors.New("Invalid format")
)

func init() {
	sparser.Register(".json", func() (sparser.Parser, error) {
		return Parser{}, nil
	})
}

const (
	TP_EOF = iota
	TP_EOF_UNEXPECTED
	TP_ERROR
	TP_STRING
	TP_NUMBER
	TP_KEYWORD
	TP_COMMA
	TP_COLON
	TP_OBJECT_START
	TP_OBJECT_END
	TP_ARRAY_START
	TP_ARRAY_END
)

type JsonScanner struct {
	rs          *scanner.Scanner
	statusStack villa.IntSlice
	out         chan Part
	stop        <-chan struct{}
}

func isHexadecimal(r rune) bool {
	return r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\r' || unicode.IsSpace(r)
}

func skipWhitespaces(s *scanner.Scanner) {
	for s.Peek() != scanner.EOF && isWhitespace(s.Peek()) {
		s.Next()
	}
}

type Part struct {
	tp         int
	start, end scanner.Position
}

func output(out chan Part, stop villa.Stop, tp int, start, end scanner.Position) (toStop bool) {
	part := Part{
		tp:    tp,
		start: start,
		end:   end,
	}

	select {
	case out <- part:
		return tp == TP_EOF || tp == TP_EOF_UNEXPECTED || tp == TP_ERROR
	case <-stop:
		return true
	}
}

func scanString(s *scanner.Scanner, out chan Part, stop villa.Stop) (toStop bool) {
	start := s.Pos()
	// start quote
	if r := s.Next(); r == scanner.EOF {
		return output(out, stop, TP_EOF_UNEXPECTED, start, s.Pos())
	} else if r != '"' {
		return output(out, stop, TP_ERROR, start, s.Pos())
	}

	// body
	for s.Peek() != '"' {
		if r := s.Next(); r == scanner.EOF {
			return output(out, stop, TP_EOF_UNEXPECTED, start, s.Pos())
		} else if r == '\\' {
			switch s.Next() {
			case scanner.EOF:
				return output(out, stop, TP_EOF_UNEXPECTED, start, s.Pos())
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				// just ok
			case 'u':
				for i := 0; i < 4; i++ {
					r := s.Next()
					if r == scanner.EOF {
						return output(out, stop, TP_EOF_UNEXPECTED, start, s.Pos())
					}

					if !isHexadecimal(r) {
						return output(out, stop, TP_ERROR, start, s.Pos())
					}
				}
			default:
				return output(out, stop, TP_ERROR, start, s.Pos())
			}
		}
	}
	// end quote
	s.Next()
	return output(out, stop, TP_STRING, start, s.Pos())
}

func scanNumber(s *scanner.Scanner, out chan Part, stop villa.Stop) (toStop bool) {
	start := s.Pos()
	if s.Peek() == '-' {
		s.Next()
	}
	if r := s.Next(); r == scanner.EOF {
		return output(out, stop, TP_EOF_UNEXPECTED, start, s.Pos())
	} else if !isDigit(r) {
		return output(out, stop, TP_ERROR, start, s.Pos())
	} else if r != 0 {
		for isDigit(s.Peek()) {
			s.Next()
		}
	}

	if s.Peek() == '.' {
		s.Next()

		if r := s.Next(); r == scanner.EOF {
			return output(out, stop, TP_EOF_UNEXPECTED, start, s.Pos())
		} else if !isDigit(r) {
			return output(out, stop, TP_ERROR, start, s.Pos())
		}
		for isDigit(s.Peek()) {
			s.Next()
		}
	}

	if s.Peek() == 'e' || s.Peek() == 'E' {
		s.Next()
		if s.Peek() == '+' || s.Peek() == '-' {
			s.Next()
		}

		if r := s.Next(); r == scanner.EOF {
			return output(out, stop, TP_EOF_UNEXPECTED, start, s.Pos())
		} else if !isDigit(r) {
			return output(out, stop, TP_ERROR, start, s.Pos())
		}
		for isDigit(s.Peek()) {
			s.Next()
		}
	}

	return output(out, stop, TP_NUMBER, start, s.Pos())
}

func scanWord(s *scanner.Scanner, out chan Part, stop villa.Stop, word []rune) (toStop bool) {
	start := s.Pos()
	for i := 1; i < len(word); i++ {
		if r := s.Next(); r == scanner.EOF {
			return output(out, stop, TP_EOF_UNEXPECTED, start, s.Pos())
		} else if r != word[i] {
			return output(out, stop, TP_ERROR, start, s.Pos())
		}
	}
	return output(out, stop, TP_KEYWORD, start, s.Pos())
}

func scanKeyword(s *scanner.Scanner, out chan Part, stop villa.Stop) (toStop bool) {
	start := s.Pos()
	switch s.Next() {
	case scanner.EOF:
		return output(out, stop, TP_EOF_UNEXPECTED, start, s.Pos())
	case 't':
		return scanWord(s, out, stop, []rune("true"))
	case 'f':
		return scanWord(s, out, stop, []rune("false"))
	case 'n':
		return scanWord(s, out, stop, []rune("null"))
	}
	return output(out, stop, TP_ERROR, start, s.Pos())
}

func scanRune(s *scanner.Scanner, out chan Part, stop villa.Stop, tp int, exp rune) (toStop bool) {
	start := s.Pos()
	if r := s.Next(); r == scanner.EOF {
		return output(out, stop, TP_EOF_UNEXPECTED, start, s.Pos())
	} else if r != exp {
		return output(out, stop, TP_ERROR, start, s.Pos())
	}
	return output(out, stop, tp, start, s.Pos())
}

func scanValue(s *scanner.Scanner, out chan Part, stop villa.Stop) (toStop bool) {
	skipWhitespaces(s)
	start := s.Pos()
	switch s.Peek() {
	case scanner.EOF:
		return output(out, stop, TP_EOF_UNEXPECTED, start, s.Pos())
	case '"':
		return scanString(s, out, stop)
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return scanNumber(s, out, stop)
	case 't', 'f', 'n':
		return scanKeyword(s, out, stop)
	case '{':
		return scanObject(s, out, stop)
	case '[':
		return scanArray(s, out, stop)
	}
	return output(out, stop, TP_ERROR, start, s.Pos())
}

func scanObject(s *scanner.Scanner, out chan Part, stop villa.Stop) (toStop bool) {
	if scanRune(s, out, stop, TP_OBJECT_START, '{') {
		return true
	}

	skipWhitespaces(s)
	if s.Peek() != '}' {
		for {
			if scanString(s, out, stop) {
				return true
			}

			skipWhitespaces(s)
			if scanRune(s, out, stop, TP_COLON, ':') {
				return true
			}

			skipWhitespaces(s)
			if scanValue(s, out, stop) {
				return true
			}

			skipWhitespaces(s)
			if s.Peek() != ',' {
				break
			}

			if scanRune(s, out, stop, TP_COMMA, ',') {
				return true
			}
			skipWhitespaces(s)
		}
	}

	return scanRune(s, out, stop, TP_OBJECT_END, '}')
}

func scanArray(s *scanner.Scanner, out chan Part, stop villa.Stop) (toStop bool) {
	if scanRune(s, out, stop, TP_ARRAY_START, '[') {
		return true
	}

	skipWhitespaces(s)
	if s.Peek() != ']' {
		for {
			if scanValue(s, out, stop) {
				return true
			}

			skipWhitespaces(s)
			if scanRune(s, out, stop, TP_COMMA, ',') {
				return true
			}

			skipWhitespaces(s)
		}
	}

	return scanRune(s, out, stop, TP_ARRAY_END, ']')
}

func parse(src []byte, out chan Part, stop villa.Stop) {
	s := &scanner.Scanner{
		Error: func(s *scanner.Scanner, msg string) {
			fmt.Println("Error", msg)
		},
	}
	s.Init(bytes.NewBuffer(src))
	s.Mode = 0

	if scanValue(s, out, stop) {
		return
	}
	output(out, stop, TP_EOF, s.Pos(), s.Pos())
}

func makeRange(start, end scanner.Position) sparser.Range {
	return sparser.Range{
		MinOffs: start.Offset,
		MaxOffs: end.Offset - 1,
		MinLine: start.Line,
		MaxLine: end.Line,
	}
}

func (Parser) Parse(in io.Reader, rcvr sparser.Receiver) error {
	src, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	stop := villa.NewStop()
	defer stop.Stop()

	out := make(chan Part)
	go parse(src, out, stop)

	var keyStart scanner.Position

	var types villa.IntSlice

loop:
	for {
		part := <-out
		switch part.tp {
		case TP_EOF:
			break loop
		case TP_EOF_UNEXPECTED:
			return EOF_UNEXPECTED
		case TP_ERROR:
			return InvalidFormat

		case TP_OBJECT_START:
			if len(types) > 0 {
				switch types[len(types)-1] {
				case TP_ARRAY_START:
					rg := makeRange(part.start, part.end)
					if err := rcvr.StartLevel(src, &rg); err != nil {
						return err
					}
				case TP_COLON:
					rg := makeRange(keyStart, part.end)
					if err := rcvr.StartLevel(src, &rg); err != nil {
						return err
					}
					types[len(types)-1] = TP_OBJECT_START
				}
			} else {
				rg := makeRange(part.start, part.end)
				if err := rcvr.StartLevel(src, &rg); err != nil {
					return err
				}
			}

			types.Add(TP_OBJECT_START)

		case TP_OBJECT_END, TP_ARRAY_END:
			rg := makeRange(part.start, part.end)
			if err := rcvr.EndLevel(src, &rg); err != nil {
				return err
			}

			types.Pop()
		case TP_STRING:
			if len(types) > 0 {
				switch types[len(types)-1] {
				case TP_OBJECT_START:
					keyStart = part.start
				case TP_ARRAY_START:
				case TP_COLON:
					rg := makeRange(keyStart, part.end)
					if err := rcvr.FinalBlock(src, &rg); err != nil {
						return err
					}
					types[len(types)-1] = TP_OBJECT_START
				}
			} else {
				rg := makeRange(part.start, part.end)
				if err := rcvr.FinalBlock(src, &rg); err != nil {
					return err
				}
			}
		case TP_COLON:
			types[len(types)-1] = TP_COLON
		case TP_ARRAY_START:
			if len(types) > 0 {
				switch types[len(types)-1] {
				case TP_ARRAY_START:
					rg := makeRange(part.start, part.end)
					if err := rcvr.StartLevel(src, &rg); err != nil {
						return err
					}
				case TP_COLON:
					rg := makeRange(keyStart, part.end)
					if err := rcvr.StartLevel(src, &rg); err != nil {
						return err
					}
					types[len(types)-1] = TP_OBJECT_START
				}
				types.Add(TP_ARRAY_START)
			}
		case TP_COMMA:
			rg := makeRange(part.start, part.end)
			if err := rcvr.FinalBlock(src, &rg); err != nil {
				return err
			}
		default:
			if len(types) > 0 {
				switch types[len(types)-1] {
				case TP_ARRAY_START:
					rg := makeRange(part.start, part.end)
					if err := rcvr.FinalBlock(src, &rg); err != nil {
						return err
					}
				case TP_COLON:
					rg := makeRange(keyStart, part.end)
					if err := rcvr.FinalBlock(src, &rg); err != nil {
						return err
					}
					types[len(types)-1] = TP_OBJECT_START
				}
			} else {
				rg := makeRange(part.start, part.end)
				if err := rcvr.FinalBlock(src, &rg); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
