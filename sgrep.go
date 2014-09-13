package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sgrep/grep"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: sgrep <pattern> [files]\n")
		flag.PrintDefaults()
	}
}

func printUsage() {
	flag.Usage()
	os.Exit(1)
}

func main() {
	pExt := flag.String("ext", "", "Specify the extension. If not specified, extract from filename")

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
	}

	pat := flag.Arg(0)
	fns := villa.Paths(args[1:]...)
	if *pExt != "" && !strings.HasPrefix(*pExt, ".") {
		*pExt = "." + *pExt
	}

	re := regexp.MustCompilePOSIX(pat)

	if len(fns) > 0 {
		for _, fn := range fns {
			grep.Grep(re, fn, *pExt)
		}
	} else {
		grep.Grep(re, "", *pExt)
	}
}
