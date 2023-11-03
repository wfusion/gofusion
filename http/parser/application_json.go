package parser

import (
	"io"
	"reflect"

	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

type ApplicationJsonParser struct{}

func (a *ApplicationJsonParser) PreParse(args map[string]string) error {
	return nil
}

func (a *ApplicationJsonParser) Parse(src io.Reader, dst reflect.Value) (err error) {
	if err = json.NewDecoder(src).Decode(dst.Addr().Interface()); err != nil {
		return malformedRequest(err.Error())
	}

	return
}
