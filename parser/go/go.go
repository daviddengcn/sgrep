package goparser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"

	"github.com/daviddengcn/sgrep/parser"
)

var goParser GoParser

func init() {
	sparser.Register(".go", func() (sparser.Parser, error) {
		return goParser, nil
	})
}

type GoParser struct{}

func rangeOfPos(fs *token.FileSet, min, max token.Pos) sparser.Range {
	minPosition, maxPosition := fs.Position(min), fs.Position(max)
	return sparser.Range{
		MinOffs: minPosition.Offset,
		MaxOffs: maxPosition.Offset,
		MinLine: minPosition.Line,
		MaxLine: maxPosition.Line,
	}
}

func maxOfFieldLists(fl *ast.FieldList) token.Pos {
	if fl == nil {
		return token.NoPos
	}

	if fl.Closing.IsValid() {
		return fl.Closing
	}
	return fl.List[len(fl.List)-1].End() - 1
}

func (GoParser) Parse(in io.Reader, rcvr sparser.Receiver) error {
	src, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", string(src), 0)
	if err != nil {
		return err
	}

	rgPackage := rangeOfPos(fs, f.Package, f.Name.End()-1)
	if err := rcvr.StartLevel(src, &rgPackage); err != nil {
		return err
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			endOfFunc := token.NoPos
			if d.Body != nil {
				endOfFunc = d.Body.Lbrace
			}
			if !endOfFunc.IsValid() {
				endOfFunc = maxOfFieldLists(d.Type.Results)
			}
			if !endOfFunc.IsValid() {
				endOfFunc = maxOfFieldLists(d.Type.Params)
			}
			header := rangeOfPos(fs, d.Type.Func, endOfFunc)
			var body *sparser.Range
			if d.Body != nil && len(d.Body.List) > 0 {
				list := d.Body.List
				r := rangeOfPos(fs, list[0].Pos(), list[len(list)-1].End()-1)
				body = &r
			}
			var footer *sparser.Range
			if d.Body != nil {
				r := rangeOfPos(fs, d.Body.Rbrace, d.Body.Rbrace)
				footer = &r
			}
			rcvr.FinalBlock(src, &header, body, footer)
		default:
			body := rangeOfPos(fs, d.Pos(), d.End()-1)
			rcvr.FinalBlock(src, nil, &body, nil)
		}
	}

	if err := rcvr.EndLevel(src, nil); err != nil {
		return err
	}
	return nil
}
