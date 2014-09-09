package txt

import (
	"io"
	"io/ioutil"
	"math"

	"github.com/daviddengcn/sgrep/parser"
)

type Parser struct {}

func init() {
	sparser.Register(".txt", func() (sparser.Parser, error) {
		return Parser{}, nil
	})
}

func (Parser) Parse(in io.Reader, rcvr sparser.Receiver) error {
	src, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}
	
	rcvr.FinalBlock(src, &sparser.Range {
		MinOffs: 0,
		MaxOffs: len(src) - 1,
		MinLine: 1,
		// grep package will handle this correctly
		MaxLine: int(math.MaxInt32),
	})
	return  nil
}