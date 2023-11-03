package encode

import (
	"fmt"
	"io"
	"os"

	"github.com/wfusion/gofusion/common/fus/util"

	"github.com/wfusion/gofusion/common/utils"
)

const (
	streamBufferSize = 16 * 1024 // 16kb
)

func Out(reader io.Reader, isStream bool) (writer io.Writer, err error) {
	switch {
	case opt.output == "":
		return nil, printStdout(reader, isStream)
	default:
		return writeFile(reader, isStream)
	}
}

func writeFile(reader io.Reader, isStream bool) (writer io.Writer, err error) {
	writer, err = os.Create(opt.output)
	if err != nil || isStream {
		return
	}

	buf, cb := utils.BytesPool.Get(streamBufferSize)
	defer cb()

	_, err = io.CopyBuffer(writer, reader, buf)
	return
}

func printStdout(reader io.Reader, isStream bool) (err error) {
	if isStream {
		fmt.Printf("output(stream):\n")
		_, err = io.Copy(os.Stdout, reader)
		return
	}

	bs, err := io.ReadAll(reader)
	if err != nil {
		return
	}
	util.PrintOutput(bs)

	return
}
