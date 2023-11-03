package parser

import (
	"fmt"
	"io"
	"reflect"

	"github.com/gin-gonic/gin"
)

type Parser interface {
	PreParse(args map[string]string) error
	Parse(src io.Reader, dst reflect.Value) error
}

func malformedRequest(msg string) error {
	return fmt.Errorf("malformed request %s", msg)
}

func unsupportedContentType(typ string) error {
	return fmt.Errorf("unsupported content-type %s", typ)
}

var (
	parserMap = map[string]reflect.Type{
		gin.MIMEJSON:              reflect.TypeOf((*ApplicationJsonParser)(nil)),
		gin.MIMEPOSTForm:          reflect.TypeOf((*ApplicationFormUrlencodedParser)(nil)),
		gin.MIMEMultipartPOSTForm: reflect.TypeOf((*MultipartFormDataParser)(nil)),
	}
)

func GetByContentType(typ string) (parser Parser, err error) {
	parserType, ok := parserMap[typ]
	if !ok {
		return nil, unsupportedContentType(typ)
	}

	return reflect.New(parserType.Elem()).Interface().(Parser), nil
}
