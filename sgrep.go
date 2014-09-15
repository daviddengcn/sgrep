package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/daviddengcn/go-ljson-conf"
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

func removeLeadingDot(ext string) string {
	if ext == "" {
		return ""
	}
	if strings.HasPrefix(ext, ".") {
		ext = ext[1:]
	}
	return ext
}

func findExtAlias(aliases map[string]string, ext string) string {
	ext = removeLeadingDot(ext)
	if newExt, ok := aliases[ext]; ok {
		return newExt
	}
	return ext
}

func loadExtAlias() map[string]string {
	conf, _ := ljconf.Load(".sgrep.json")
	aMap := conf.Object("aliases", nil)
	res := make(map[string]string)
	for dst, aliases := range aMap {
		dst = removeLeadingDot(dst)

		if lst, ok := aliases.([]interface{}); ok {
			for _, alias := range lst {
				src := removeLeadingDot(fmt.Sprint(alias))
				res[src] = dst
			}
		} else {
			src := removeLeadingDot(fmt.Sprint(aliases))
			res[src] = dst
		}
	}
	return res
}

func main() {
	aliases := loadExtAlias()

	pExt := flag.String("ext", "", "Specify the extension. If not specified, extract from filename")

	flag.Parse()

	*pExt = findExtAlias(aliases, removeLeadingDot(*pExt))

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
	}

	pat := flag.Arg(0)
	fns := villa.Paths(args[1:]...)

	re := regexp.MustCompilePOSIX(pat)

	if len(fns) > 0 {
		for _, fn := range fns {
			ext := *pExt
			if ext == "" {
				ext = findExtAlias(aliases, removeLeadingDot(fn.Ext()))
			}

			grep.Grep(re, fn, ext)
		}
	} else {
		grep.Grep(re, "", *pExt)
	}
}
