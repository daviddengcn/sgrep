package parser

import (
	"strings"
	"testing"

	"github.com/daviddengcn/go-assert"
)

func TestLines(t *testing.T) {
	lines := Lines{
		HeaderField: []string{
			"lines {",
		},
		BodyField: []string{
			"  Hello,",
			"  World!",
		},
		FooterField: []string{
			"} // lines",
		},
	}
	results := []string{}

	isFinal, header, err := lines.Start()
	assert.NoError(t, err)
	assert.Equals(t, "isFinal", true, isFinal)
	results = append(results, header...)

	body, err := lines.Body()
	assert.NoError(t, err)
	results = append(results, body...)

	footer, err := lines.End()
	assert.NoError(t, err)
	results = append(results, footer...)

	assert.LinesEqual(t, "results", results, strings.Split(
		`lines {
  Hello,
  World!
} // lines`, "\n"))
}
