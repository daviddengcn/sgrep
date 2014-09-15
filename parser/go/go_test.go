package goparser

import (
	"fmt"
	"testing"
	
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sgrep/parser"
	"github.com/daviddengcn/go-assert"
)

func Test(t *testing.T) {
	src :=
`package example

import "testing"

type T struct {
	Field int
}

func Foo() {
	Hello
}`

	exp :=
`1: S package example
3: F import "testing"
5: S type
5: F T struct {
	Field int

7: E }
9: S func Foo() {
10: F Hello
11: E }
`

	act := ""
	rcvr := sparser.ReceiverFunc {
		StartLevelFunc: func(buffer []byte, header sparser.Range) error {
			if header.IsEmpty() {
				return nil
			}
			act += fmt.Sprintf("%d: ", header.MinLine)
			act += "S " + string(buffer[header.MinOffs:header.MaxOffs+1]) + "\n"
			return nil
		},
		
		FinalBlockFunc: func(buffer []byte, body sparser.Range) error {
			if body.IsEmpty() {
				return nil
			}
			act += fmt.Sprintf("%d: ", body.MinLine)
			act += "F " + string(buffer[body.MinOffs:body.MaxOffs+1]) + "\n"
			return nil
		},
		
		EndLevelFunc: func(buffer []byte, footer sparser.Range) error {
			if footer.IsEmpty() {
				return nil
			}
			act += fmt.Sprintf("%d: ", footer.MinLine)
			act += "E " + string(buffer[footer.MinOffs:footer.MaxOffs+1]) + "\n"
			return nil
		},
	}
	
	srcBytes := villa.ByteSlice(src)
	assert.NoError(t, Parser{}.Parse(&srcBytes, rcvr))
	
	assert.TextEquals(t, "act", act, exp)
}
