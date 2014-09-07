package sparser

import (
	"errors"
	"io"
	"testing"

	"github.com/daviddengcn/go-assert"
)

type emptyParser struct{}

func (*emptyParser) Parse(in io.Reader, rcvr Receiver) error {
	return nil
}

func TestRegister(t *testing.T) {
	_, err := New(".TestRegister")
	assert.Equals(t, "err", err.Deepest(), UnknownExtension)

	p := &emptyParser{}
	Register(".TestRegister", func() (Parser, error) {
		return p, nil
	})

	ps, err := New(".TestRegister")
	assert.NoErrorf(t, "New(.TestRegister): %v", err)
	assert.Equals(t, "Parser", ps, p)

	myErr := errors.New("myerr")
	Register(".TestRegisterError", func() (Parser, error) {
		return nil, myErr
	})
	ps, err = New(".TestRegisterError")
	assert.Equals(t, "ps", ps, nil)
	assert.Equals(t, "err", err.Deepest(), myErr)
}
