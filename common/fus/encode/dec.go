package encode

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/encode"
)

func runDec(cmd *cobra.Command, args []string) (err error) {
	opts := make([]utils.OptionExtender, 0, 3)
	for _, encodedType := range orders {
		switch encodedType {
		case encode.EncodedTypeCipher:
			opts = append(opts, encode.Cipher(opt.cipherAlgo, opt.cipherMode, opt.key, opt.iv))
		case encode.EncodedTypeCompress:
			opts = append(opts, encode.Compress(opt.compressAlgo))
		case encode.EncodedTypeEncode:
			opts = append(opts, encode.Encode(opt.encodeAlgo))
		}
	}

	reader, isStream, err := In(args[0])
	if err != nil {
		return
	}

	var dstReader io.Reader
	if !isStream {
		var dst []byte
		src, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		rvs := make([]utils.OptionExtender, 0, len(opts))
		for i := len(opts) - 1; i >= 0; i-- {
			rvs = append(rvs, opts[i])
		}
		if _, err = utils.Catch(func() { dst = utils.Must(encode.From(src).Decode(rvs...).ToBytes()) }); err != nil {
			return errors.Cause(err)
		}
		dstReader = bytes.NewBuffer(dst)
	}

	w, err := Out(dstReader, isStream)
	if err != nil {
		return
	}
	if isStream && w != nil {
		if w == nil {
			return errors.New("stream destination unset")
		}
		defer utils.CloseAnyway(w)
		_, err = encode.NewCodecStream(opts...).Decode(w, reader)
	}
	return errors.Cause(err)
}
