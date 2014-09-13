package txt

import (
	"fmt"
	"testing"

	"github.com/daviddengcn/go-assert"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sgrep/parser"
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
		`1: F {
	"hello": "world",
	"numbers": [
		true,
		{ "go": 1
		},
		4
	]
}
`

	act := ""
	rcvr := sparser.ReceiverFunc{
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
