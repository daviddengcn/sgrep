package goparser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"

	"github.com/daviddengcn/sgrep/parser"
)

type Parser struct{}

func init() {
	sparser.Register("go", func() (sparser.Parser, error) {
		return Parser{}, nil
	})
}

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

func (Parser) Parse(in io.Reader, rcvr sparser.Receiver) error {
	src, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", string(src), 0)
	if err != nil {
		return err
	}

	if err := rcvr.StartLevel(src, rangeOfPos(fs, f.Package, f.Name.End()-1)); err != nil {
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
			var body sparser.Range
			if d.Body != nil && len(d.Body.List) > 0 {
				list := d.Body.List
				body = rangeOfPos(fs, list[0].Pos(), list[len(list)-1].End()-1)
			}
			var footer sparser.Range
			if d.Body != nil {
				footer = rangeOfPos(fs, d.Body.Rbrace, d.Body.Rbrace)
			}
			if err := rcvr.StartLevel(src, header); err != nil {
				return err
			}
			if err := rcvr.FinalBlock(src, body); err != nil {
				return err
			}
			if err := rcvr.EndLevel(src, footer); err != nil {
				return err
			}
		case *ast.GenDecl:
			if d.Lparen.IsValid() && len(d.Specs) > 0 {
				header := rangeOfPos(fs, d.TokPos, d.Lparen)
				body := rangeOfPos(fs, d.Specs[0].Pos(), d.Specs[len(d.Specs)-1].End() - 1)
				footer := rangeOfPos(fs, d.Rparen, d.Rparen)
				
				if err := rcvr.StartLevel(src, header); err != nil {
					return err
				}
				if err := rcvr.FinalBlock(src, body); err != nil {
					return err
				}
				if err := rcvr.EndLevel(src, footer); err != nil {
					return err
				}
			} else if len(d.Specs) == 1 && d.Tok != token.IMPORT {
				header := rangeOfPos(fs, d.TokPos, d.TokPos + token.Pos(len(d.Tok.String()) - 1))
				body := rangeOfPos(fs, d.Specs[0].Pos(), d.Specs[len(d.Specs)-1].End() - 2)
				footer := rangeOfPos(fs, d.Specs[0].End() - 1, d.Specs[0].End() - 1)
				
				if err := rcvr.StartLevel(src, header); err != nil {
					return err
				}
				if err := rcvr.FinalBlock(src, body); err != nil {
					return err
				}
				if err := rcvr.EndLevel(src, footer); err != nil {
					return err
				}
			} else {
				if err := rcvr.FinalBlock(src, rangeOfPos(fs, d.Pos(), d.End()-1)); err != nil {
					return err
				}
			}
		default:
			if err := rcvr.FinalBlock(src, rangeOfPos(fs, d.Pos(), d.End()-1)); err != nil {
				return err
			}
		}
	}

	if err := rcvr.EndLevel(src, sparser.Range{}); err != nil {
		return err
	}
	return nil
}
