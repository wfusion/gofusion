package parser

import (
	"io"
	"net/url"
	"reflect"
)

type ApplicationFormUrlencodedParser struct{}

func (a *ApplicationFormUrlencodedParser) PreParse(args map[string]string) error {
	return nil
}

func (a *ApplicationFormUrlencodedParser) Parse(src io.Reader, dst reflect.Value) (err error) {
	body, err := io.ReadAll(src)
	if err != nil {
		return
	}

	vals, err := url.ParseQuery(string(body))
	if err != nil {
		return
	}

	return MapFormByTag(dst.Addr().Interface(), vals, "json")
}
