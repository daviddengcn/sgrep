package xml

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"text/scanner"

	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sgrep/parser"
)

type Parser struct{}

func init() {
	sparser.Register(".xml", func() (sparser.Parser, error) {
		return Parser{}, nil
	})
}

func isWhiteSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == '\n'
}

const (
	TP_NONE = iota
	TP_FINAL
	TP_START
	TP_END
)

func scanTo1(s *scanner.Scanner, target rune) bool {
	for {
		switch s.Next() {
		case scanner.EOF:
			return false
		case target:
			return true
		}
	}
}

func scanTo2(s *scanner.Scanner, target0, target1 rune) bool {
	for {
		switch s.Next() {
		case scanner.EOF:
			return false
		case target0:
			if s.Peek() == target1 {
				s.Next() // skip targe1
				return true
			}
		}
	}
}

func scanTo3(s *scanner.Scanner, target0, target1, target2 rune) bool {
	for {
		// match 0, 1
		if !scanTo2(s, target0, target1) {
			// EOF
			return false
		}
		switch s.Peek() {
		case scanner.EOF:
			s.Next()
			return false
		case target2:
			// matched 2, found
			s.Next()
			return true
		case target1:
			if target0 == target1 {
			loop:
				for {
					switch s.Next() {
					case scanner.EOF:
						return false
					case target2:
						return true
					case target0:
						// keep scanning
					default:
						// not found, go to out loop
						break loop
					}
				}
			}
		}
	}
}

func rangeFromStart(s *scanner.Scanner, start scanner.Position) sparser.Range {
	p := s.Pos()
	return sparser.Range{
		MinOffs: start.Offset,
		MaxOffs: p.Offset,
		MinLine: start.Line,
		MaxLine: p.Line,
	}
}

func scanBlock(s *scanner.Scanner) (blockType int, rg sparser.Range, name string) {
	if s.Peek() != '<' {
		start := s.Pos()
		for s.Peek() != scanner.EOF && s.Peek() != '<' {
			s.Next()
		}
		return TP_FINAL, rangeFromStart(s, start), ""
	}
	// '<'
	lt := s.Next()
	if lt == scanner.EOF {
		return TP_NONE, sparser.Range{}, ""
	}

	start := s.Pos()
	if lt != '<' {
		// malformed, return as a final block
		return TP_FINAL, sparser.Range{
			MinOffs: start.Offset,
			MaxOffs: start.Offset,
			MinLine: start.Line,
			MaxLine: start.Line,
		}, ""
	}

	switch tp := s.Next(); tp {
	case '?':
		// PI
		// to find ?>
		scanTo2(s, '?', '>')
		p := s.Pos()
		return TP_FINAL, sparser.Range{
			MinOffs: start.Offset,
			MaxOffs: p.Offset,
			MinLine: start.Line,
			MaxLine: p.Line,
		}, ""
	case '!':
		switch s.Next() {
		case scanner.EOF:
			// malformed
		case '[':
			// <![CDATA
			// find ]]>
			scanTo3(s, ']', ']', '>')
		case '-':
			// comments
			// find -->
			scanTo3(s, '-', '-', '>')
		default:
			// Attribute-List
			// find >
			scanTo1(s, '>')
		}
		p := s.Pos()
		return TP_FINAL, sparser.Range{
			MinOffs: start.Offset,
			MaxOffs: p.Offset,
			MinLine: start.Line,
			MaxLine: p.Line,
		}, ""
	case '/':
		// end tag
		name := make([]rune, 0, 8)
		for {
			r := s.Next()
			if r == scanner.EOF || r == '>' {
				break
			}
			if isWhiteSpace(r) {
				scanTo1(s, '>')
				break
			}
			name = append(name, r)
		}
		p := s.Pos()
		if len(name) == 0 {
			// malformed
			return TP_FINAL, sparser.Range{
				MinOffs: start.Offset,
				MaxOffs: p.Offset,
				MinLine: start.Line,
				MaxLine: p.Line,
			}, ""
		}
		return TP_END, sparser.Range{
			MinOffs: start.Offset,
			MaxOffs: p.Offset,
			MinLine: start.Line,
			MaxLine: p.Line,
		}, string(name)
	case '>':
		// malformed
		p := s.Pos()
		return TP_FINAL, sparser.Range{
			MinOffs: start.Offset,
			MaxOffs: p.Offset,
			MinLine: start.Line,
			MaxLine: p.Line,
		}, ""
	default:
		// start tag
		name := []rune{tp}
		for {
			r := s.Next()
			if r == scanner.EOF || r == '>' {
				break
			}

			if r == '/' {
				if s.Peek() == '>' {
					s.Next()
					p := s.Pos()
					return TP_FINAL, sparser.Range{
						MinOffs: start.Offset,
						MaxOffs: p.Offset,
						MinLine: start.Line,
						MaxLine: p.Line,
					}, string(name)
				}
			}

			if isWhiteSpace(r) {
			loop:
				for {
					switch s.Next() {
					case scanner.EOF:
						p := s.Pos()
						return TP_FINAL, sparser.Range{
							MinOffs: start.Offset,
							MaxOffs: p.Offset,
							MinLine: start.Line,
							MaxLine: p.Line,
						}, string(name)
					case '/':
						if s.Peek() == '>' {
							s.Next()
							p := s.Pos()
							return TP_FINAL, sparser.Range{
								MinOffs: start.Offset,
								MaxOffs: p.Offset,
								MinLine: start.Line,
								MaxLine: p.Line,
							}, string(name)
						}
					case '>':
						break loop
					}
				}
				break
			}
			name = append(name, r)
		}
		p := s.Pos()
		return TP_START, sparser.Range{
			MinOffs: start.Offset,
			MaxOffs: p.Offset,
			MinLine: start.Line,
			MaxLine: p.Line,
		}, string(name)
	}
}

func lastIndexOf(stack villa.StringSlice, name string) int {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == name {
			return i
		}
	}
	return -1
}

func (Parser) Parse(in io.Reader, rcvr sparser.Receiver) error {
	src, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}
	s := &scanner.Scanner{
		Error: func(s *scanner.Scanner, msg string) {
			fmt.Println("Error", msg)
		},
	}
	s.Init(bytes.NewBuffer(src))
	s.Mode = 0

	var stack villa.StringSlice

	for s.Peek() != scanner.EOF {
		blockType, rg, name := scanBlock(s)
		switch blockType {
		case TP_FINAL:
			if err := rcvr.FinalBlock(src, &rg); err != nil {
				return err
			}
		case TP_START:
			if len(name) > 0 {
				if err := rcvr.StartLevel(src, &rg); err != nil {
					return err
				}
				stack.Add(name)
			} else {
				if err := rcvr.FinalBlock(src, &rg); err != nil {
					return err
				}
			}
		case TP_END:
			if len(name) > 0 {
				p := lastIndexOf(stack, name)
				if p < 0 {
					if err := rcvr.FinalBlock(src, &rg); err != nil {
						return err
					}
				} else {
					for len(stack) < p+1 {
						if err := rcvr.EndLevel(src, nil); err != nil {
							return err
						}
						stack.Pop()
					}
					if err := rcvr.EndLevel(src, &rg); err != nil {
						return err
					}
				}
			} else {
				if err := rcvr.FinalBlock(src, &rg); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
