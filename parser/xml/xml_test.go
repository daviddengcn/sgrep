package xml

import (
	"fmt"
	"testing"

	"github.com/daviddengcn/go-assert"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sgrep/parser"
)

func TestBasic(t *testing.T) {
	src :=
		`<?xml version="1.0" encoding="UTF-8"?>
<go>
	<hello>come on</hello><br/> <!-- Hello -->
	<data><![CDATANew
 Line]]></data>
</go>`

	exp :=
		`1: F <?xml version="1.0" encoding="UTF-8"?>
2: S <go>
3: S <hello>
3: F come on
3: E </hello>
3: F <br/>
4: S <data>
4: F <![CDATANew
 Line]]>
5: E </data>
6: E </go>
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
