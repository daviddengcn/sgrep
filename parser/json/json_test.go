package json

import (
	"fmt"
	"testing"
	
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sgrep/parser"
	"github.com/daviddengcn/go-assert"
)

func Test(t *testing.T) {
	src :=
`{
	"hello": "world",
	"numbers": [
		true,
		{ "go": 1
		},
		4
	]
}`

	exp :=
`1: S {
2: F "hello": "world"
2: F ,
3: S "numbers": [
4: F true
4: F ,
5: S {
5: F "go": 1
6: E }
6: F ,
7: F 4
8: E ]
9: E }
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
