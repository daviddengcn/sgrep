package grep

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"

	"github.com/daviddengcn/go-colortext"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sgrep/parser"
	_ "github.com/daviddengcn/sgrep/parser/go"
	"github.com/daviddengcn/sgrep/parser/indent"
	_ "github.com/daviddengcn/sgrep/parser/json"
	_ "github.com/daviddengcn/sgrep/parser/xml"
)

func markAndPrint(ln int, re *regexp.Regexp, line []byte) {
	locs := re.FindAllIndex(line, -1)
	if len(locs) > 0 {
		fmt.Printf("%4d: ", ln)
	} else {
		fmt.Print("      ")
	}
	p := 0
	for _, loc := range locs {
		if loc[0] > p {
			os.Stdout.Write(line[p:loc[0]])
		}
		ct.ChangeColor(ct.Green, true, ct.None, false)
		os.Stdout.Write(line[loc[0]:loc[1]])
		ct.ResetColor()
		p = loc[1]
	}
	if p < len(line) {
		os.Stdout.Write(line[p:])
	}
	fmt.Println()
}

type LevelInfo struct {
	headerPrinted bool
	headerBuffer  []byte
	header        sparser.Range

	hiddenChidren int
	hiddenLines   int
	found         bool
}

type Receiver struct {
	fn        villa.Path
	fnPrinted bool
	re        *regexp.Regexp
	// 1-based
	maxPrintedLine int
	infos          []LevelInfo
}

func (rcvr *Receiver) beforeBody(level int) {
	info := rcvr.infos[level]
	if !info.headerPrinted {
		rcvr.beforeBody(level - 1)

		// Print the header
		rcvr.showRange(info.headerBuffer, info.header)
		info.headerPrinted = true
	} else if !rcvr.fnPrinted {
		fmt.Println(rcvr.fn)
		rcvr.fnPrinted = true
	}
}

func (rcvr *Receiver) StartLevel(buffer []byte, header sparser.Range) error {
	rcvr.infos = append(rcvr.infos, LevelInfo{
		headerBuffer: buffer,
		header:       header,
	})
	info := &rcvr.infos[len(rcvr.infos)-1]

	if findInBuffer(rcvr.re, buffer, header) {
		info.found = true
		rcvr.beforeBody(len(rcvr.infos) - 1)
	}

	return nil
}

func (rcvr *Receiver) EndLevel(buffer []byte, footer sparser.Range) error {
	info := rcvr.infos[len(rcvr.infos)-1]

	info.found = info.found || findInBuffer(rcvr.re, buffer, footer)
	if info.found {
		rcvr.beforeBody(len(rcvr.infos) - 1)
		rcvr.showRange(buffer, footer)

		rcvr.infos[len(rcvr.infos)-2].found = true
	}

	rcvr.infos = rcvr.infos[:len(rcvr.infos)-1]
	return nil
}

func findInBuffer(re *regexp.Regexp, buffer []byte, r sparser.Range) bool {
	if r.IsEmpty() {
		return false
	}
	return re.FindIndex(buffer[r.MinOffs:r.MaxOffs+1]) != nil
}

func relocateLineStart(buffer []byte, offs int) int {
	for ; offs > 0 && buffer[offs-1] != '\n'; offs-- {
	}
	return offs
}

func findLineEnd(buffer []byte, offs int) int {
	l := bytes.IndexByte(buffer[offs:], '\n')
	if l < 0 {
		// last line in buffer
		l = len(buffer) - offs
	}
	return offs + l
}

func (rcvr *Receiver) markAndPrint(line int, buffer []byte) {
	markAndPrint(line, rcvr.re, buffer)
}

// Returns the start of next line
func (rcvr *Receiver) showLine(buffer []byte, offs int, line int) int {
	end := findLineEnd(buffer, offs)

	if line > rcvr.maxPrintedLine {
		rcvr.markAndPrint(line, buffer[offs:end])
		rcvr.maxPrintedLine = line
	}

	if end < len(buffer) {
		// move over \n
		end++
	}
	return end
}

func (rcvr *Receiver) showRange(buffer []byte, r sparser.Range) {
	if r.IsEmpty() {
		return
	}

	offs := relocateLineStart(buffer, r.MinOffs)
	for line := r.MinLine; line <= r.MaxLine; line++ {
		offs = rcvr.showLine(buffer, offs, line)
	}
}

func (rcvr *Receiver) FinalBlock(buffer []byte, body sparser.Range) error {
	if !findInBuffer(rcvr.re, buffer, body) {
		// no match, skipped
		return nil
	}

	rcvr.beforeBody(len(rcvr.infos) - 1)
	rcvr.infos[len(rcvr.infos)-1].found = true

	if !body.IsEmpty() {
		offs := body.MinOffs
		for line := body.MinLine; line <= body.MaxLine; line++ {
			end := findLineEnd(buffer, offs)

			if rcvr.re.FindIndex(buffer[offs:end]) != nil {
				rcvr.showLine(buffer, relocateLineStart(buffer, offs), line)
			}

			if end >= len(buffer) {
				break
			}
			offs = end + 1
		}
	}
	return nil
}

// ext doesn't start with '.'
func Grep(re *regexp.Regexp, fn villa.Path, ext string) {
	isIndent := false
	var err error
	p, err := sparser.New(ext)
	if err != nil {
		p = indent.Parser{}
		isIndent = true
	}

	var f io.Reader
	if fn == "" {
		f = os.Stdin
	} else {
		ff, err := fn.Open()
		if err != nil {
			log.Fatalf("Open file %v failed: %v", fn, err)
		}
		defer ff.Close()
		f = ff
	}

	receiver := Receiver{
		fn: fn,
		re: re,
		infos: []LevelInfo{
			LevelInfo{
				headerPrinted: true,
			},
		},
	}

	if err := p.Parse(f, &receiver); err != nil {
		if !isIndent && fn != "" {
			// Try use indent parser
			p = indent.Parser{}
			iReceiver := Receiver{
				fn: fn,
				re: re,
				infos: []LevelInfo{
					LevelInfo{
						headerPrinted: true,
					},
				},
				maxPrintedLine: receiver.maxPrintedLine,
				fnPrinted: receiver.fnPrinted,
			}

			ff, err := fn.Open()
			if err != nil {
				log.Fatalf("Open file %v failed: %v", fn, err)
			}
			defer ff.Close()
			f = ff

			if err := p.Parse(f, &iReceiver); err == nil {
				return
			}
		}
		log.Fatalf("Parse failed: %v", err)
	}
}
