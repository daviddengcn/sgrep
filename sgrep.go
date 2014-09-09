package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/daviddengcn/go-colortext"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sgrep/parser"
	_ "github.com/daviddengcn/sgrep/parser/go"
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

/*
func (info *GrepInfo) beforeHeader(parents []*GrepInfo) {
	if len(parents) > 0 {
		parents[len(parents)-1].beforeBody(parents[:len(parents)-1])
	}
}
func (info *GrepInfo) beforeBody(parents []*GrepInfo) {
	if !info.headerPrinted {
		info.beforeHeader(parents)

		// Print the header
		for _, line := range info.header {
			fmt.Println(line)
		}
		info.headerPrinted = true
	}
	if info.hiddenChidren > 0 {
		fmt.Println("  ...")
		//		ct.ChangeColor(ct.Yellow, false, ct.None, false)
		//		fmt.Printf("(%d blocks)\n", info.hiddenChidren)
		//		ct.ResetColor()
		info.hiddenChidren = 0
	} else if info.hiddenLines > 0 {
		fmt.Println("  ...")
		//		ct.ChangeColor(ct.Yellow, false, ct.None, false)
		//		fmt.Printf("(%d lines)\n", info.hiddenLines)
		//		ct.ResetColor()
		info.hiddenLines = 0
	}
}
*/
func foundInLines(re *regexp.Regexp, lines []string) bool {
	for _, line := range lines {
		if re.FindStringIndex(line) != nil {
			return true
		}
	}
	return false
}

/*
func grep(re *regexp.Regexp, t parser.Node, parents []*GrepInfo) (bool, error) {
	info := &GrepInfo{
		t: t,
	}
	current := append(parents, info)
	found := false

	isFinal, header, err := t.Start()
	if err != nil {
		return false, err
	}

	info.header = header

	if foundInLines(re, header) {
		info.beforeHeader(parents)

		// Print the header
		markAndPrint(re, info.header...)
		info.headerPrinted = true

		found = true
	}
	if isFinal {
		body, err := t.Body()
		if err != nil {
			return false, err
		}

		for _, line := range body {
			if re.FindStringIndex(line) != nil {
				info.beforeBody(parents)
				markAndPrint(re, line)
				found = true
			} else {
				info.hiddenLines++
			}
		}
	} else {
		for {
			node, err := t.Next()
			if err == parser.EOF {
				break
			}
			if err != nil {
				return false, err
			}

			fd, err := grep(re, node, current)
			if err != nil {
				return false, err
			}

			if fd {
				found = true
			} else {
				info.hiddenChidren++
			}
		}
	}

	footer, err := t.End()
	if !found {
		found = foundInLines(re, footer)
	}
	if found {
		info.beforeBody(parents)
		markAndPrint(re, footer...)
	}
	return found, nil
}
*/

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
	info := &rcvr.infos[len(rcvr.infos) - 1]

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
//fmt.Println("FFF", string(buffer[body.MinOffs:body.MaxOffs+1]))
		// no match, skipped
		return nil
	}

	rcvr.beforeBody(len(rcvr.infos) - 1)
	rcvr.infos[len(rcvr.infos) - 1].found = true

	if body != nil {
		offs := body.MinOffs
		for line := body.MinLine; line <= body.MaxLine; line++ {
			end := findLineEnd(buffer, offs)

			if rcvr.re.FindIndex(buffer[offs:end]) != nil {
				rcvr.showLine(buffer, relocateLineStart(buffer, offs), line)
			}

			offs = end+1
		}
	}
	return nil
}

func main() {
//	fn := villa.Path("sgrep.go")
//	pat := "int"
	fn := villa.Path("pom.xml")
	pat := "google"
	re := regexp.MustCompilePOSIX(pat)

	var err error
	p, err := sparser.New(fn.Ext())
	if err != nil {
		log.Fatalf("New Parser failed for suffix %s: %v", fn.Ext(), err)
	}

	f, err := fn.Open()
	if err != nil {
		log.Fatalf("Open file %v failed: %v", fn, err)
	}
	defer f.Close()

	receiver := Receiver{
		re: re,
		infos: []LevelInfo {
			LevelInfo{
				headerPrinted: true,
			},
		},
	}

	if err := p.Parse(f, &receiver); err != nil {
		log.Fatalf("Parsed failed: %v", err)
	}
}
