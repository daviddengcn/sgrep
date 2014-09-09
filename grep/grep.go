package grep

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"io"
	
	"github.com/daviddengcn/go-colortext"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sgrep/parser"
	_ "github.com/daviddengcn/sgrep/parser/go"
	_ "github.com/daviddengcn/sgrep/parser/xml"
	_ "github.com/daviddengcn/sgrep/parser/txt"
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
	header        *sparser.Range

	hiddenChidren int
	hiddenLines   int
	found         bool
}

type Receiver struct {
	re *regexp.Regexp
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
	}
}

func (rcvr *Receiver) StartLevel(buffer []byte, header *sparser.Range) error {
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

func (rcvr *Receiver) EndLevel(buffer []byte, footer *sparser.Range) error {
	info := rcvr.infos[len(rcvr.infos)-1]

	info.found = info.found || findInBuffer(rcvr.re, buffer, footer)
	if info.found {
		rcvr.showRange(buffer, footer)

		rcvr.infos[len(rcvr.infos)-2].found = true
	}

	rcvr.infos = rcvr.infos[:len(rcvr.infos)-1]
	return nil
}

func findInBuffer(re *regexp.Regexp, buffer []byte, r *sparser.Range) bool {
	if r == nil {
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

func (rcvr *Receiver) showRange(buffer []byte, r *sparser.Range) {
	if r == nil {
		return
	}

	offs := relocateLineStart(buffer, r.MinOffs)
	for line := r.MinLine; line <= r.MaxLine; line++ {
		offs = rcvr.showLine(buffer, offs, line)
	}
}

func (rcvr *Receiver) FinalBlock(buffer []byte, body *sparser.Range) error {
	if !findInBuffer(rcvr.re, buffer, body) {
		// no match, skipped
		return nil
	}

	rcvr.beforeBody(len(rcvr.infos) - 1)
	rcvr.infos[len(rcvr.infos)-1].found = true

	if body != nil {
		offs := body.MinOffs
		for line := body.MinLine; line <= body.MaxLine; line++ {
			end := findLineEnd(buffer, offs)
			if end >= len(buffer) {
				break
			}

			if rcvr.re.FindIndex(buffer[offs:end]) != nil {
				rcvr.showLine(buffer, relocateLineStart(buffer, offs), line)
			}

			offs = end + 1
		}
	}
	return nil
}

func Grep(re *regexp.Regexp, fn villa.Path, ext string) {
	if ext == "" {
		ext = fn.Ext()
	}
	if ext == "" {
		ext = ".txt"
	}
	
	var err error
	p, err := sparser.New(ext)
	if err != nil {
		log.Fatalf("Creating Parser failed: %v", err)
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
		re: re,
		infos: []LevelInfo{
			LevelInfo{
				headerPrinted: true,
			},
		},
	}

	if err := p.Parse(f, &receiver); err != nil {
		log.Fatalf("Parsed failed: %v", err)
	}
}
