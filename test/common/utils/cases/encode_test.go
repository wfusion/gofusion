package cases

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/cipher"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/encode"
	"github.com/wfusion/gofusion/log"

	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestEncode(t *testing.T) {
	testingSuite := &Encode{Test: testUtl.T}
	suite.Run(t, testingSuite)
}

type Encode struct {
	*testUtl.Test
}

func (t *Encode) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Encode) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Encode) TestCodecSingleCombination() {
	cases := [][]encode.EncodedType{
		{encode.EncodedTypeCipher},
		{encode.EncodedTypeCompress},
		{encode.EncodedTypeEncode},
	}
	t.runCodecCases(cases)
}

func (t *Encode) TestCodecDoubleCombinations() {
	cases := [][]encode.EncodedType{
		{encode.EncodedTypeCipher, encode.EncodedTypeCompress},
		{encode.EncodedTypeCompress, encode.EncodedTypeCipher},
		{encode.EncodedTypeCipher, encode.EncodedTypeEncode},
		{encode.EncodedTypeEncode, encode.EncodedTypeCipher},
		{encode.EncodedTypeCompress, encode.EncodedTypeEncode},
		{encode.EncodedTypeEncode, encode.EncodedTypeCompress},
	}

	t.runCodecCases(cases)
}

func (t *Encode) TestCodecTripleCombinations() {
	cases := [][]encode.EncodedType{
		{encode.EncodedTypeCipher, encode.EncodedTypeCompress, encode.EncodedTypeEncode},
		{encode.EncodedTypeCipher, encode.EncodedTypeEncode, encode.EncodedTypeCompress},
		{encode.EncodedTypeCompress, encode.EncodedTypeCipher, encode.EncodedTypeEncode},
		{encode.EncodedTypeCompress, encode.EncodedTypeEncode, encode.EncodedTypeCipher},
		{encode.EncodedTypeEncode, encode.EncodedTypeCompress, encode.EncodedTypeCipher},
		{encode.EncodedTypeEncode, encode.EncodedTypeCipher, encode.EncodedTypeCompress},
	}

	t.runCodecCases(cases)
}

func (t *Encode) TestStreamSingleCombination() {
	cases := [][]encode.EncodedType{
		{encode.EncodedTypeCipher},
		{encode.EncodedTypeCompress},
		{encode.EncodedTypeEncode},
	}

	t.runStreamCases(cases)
	t.runStreamParallelCases(cases)
}

func (t *Encode) TestStreamDoubleCombinations() {
	cases := [][]encode.EncodedType{
		{encode.EncodedTypeCipher, encode.EncodedTypeCompress},
		{encode.EncodedTypeCompress, encode.EncodedTypeCipher},
		{encode.EncodedTypeCipher, encode.EncodedTypeEncode},
		{encode.EncodedTypeEncode, encode.EncodedTypeCipher},
		{encode.EncodedTypeCompress, encode.EncodedTypeEncode},
		{encode.EncodedTypeEncode, encode.EncodedTypeCompress},
	}

	t.runStreamCases(cases)
	t.runStreamParallelCases(cases)
}

func (t *Encode) TestStreamTripleCombinations() {
	cases := [][]encode.EncodedType{
		{encode.EncodedTypeCipher, encode.EncodedTypeCompress, encode.EncodedTypeEncode},
		{encode.EncodedTypeCipher, encode.EncodedTypeEncode, encode.EncodedTypeCompress},
		{encode.EncodedTypeCompress, encode.EncodedTypeCipher, encode.EncodedTypeEncode},
		{encode.EncodedTypeCompress, encode.EncodedTypeEncode, encode.EncodedTypeCipher},
		{encode.EncodedTypeEncode, encode.EncodedTypeCompress, encode.EncodedTypeCipher},
		{encode.EncodedTypeEncode, encode.EncodedTypeCipher, encode.EncodedTypeCompress},
	}

	t.runStreamCases(cases)
	t.runStreamParallelCases(cases)
}

func (t *Encode) runCodecCases(cases [][]encode.EncodedType) {
	t.Catch(func() {
		data := t.smallRandomData()
		for _, cs := range cases {
			for _, testCase := range t.generateCodecCases(cs...) {
				t.Run(strings.Join(testCase.names, "->"), func() {
					options := testCase.extenders
					reversed := make([]utils.OptionExtender, len(options))
					for i, v := range options {
						reversed[len(options)-1-i] = v
					}

					// case: encode then decode
					encoded, err := encode.From(data).Encode(options...).ToBytes()
					t.NoError(err)
					t.NotEqualValues(data, encoded)

					codec := encode.From(encoded)
					for i := 0; i < len(reversed); i++ {
						codec = codec.Decode(reversed[i])
					}
					decoded, err := codec.ToString()
					t.NoError(err)
					t.EqualValues(data, []byte(decoded))

					// case: encoded and decode together
					actual, err := encode.
						From(data).
						Encode(options...).
						Decode(reversed...).
						ToBytes()
					t.NoError(err)
					t.EqualValues(data, actual)
				})
			}
		}
	})
}

func (t *Encode) runStreamCases(cases [][]encode.EncodedType) {
	t.Catch(func() {
		data := t.randomData()
		for _, cs := range cases {
			for _, testCase := range t.generateStreamCases(cs...) {
				t.Run(strings.Join(testCase.names, "->"), func() {
					codecStream := encode.NewCodecStream(testCase.extenders...)
					dataBuffer := bytes.NewReader(data)

					encodedBuffer := bytes.NewBuffer(nil)
					_, err := codecStream.Encode(encodedBuffer, dataBuffer)
					t.NoError(err)
					t.NotZero(encodedBuffer.Len())
					t.NotEqualValues(data, encodedBuffer.Bytes())

					decodedBuffer := bytes.NewBuffer(nil)
					_, err = codecStream.Decode(decodedBuffer, encodedBuffer)
					t.NoError(err)

					t.EqualValues(data, decodedBuffer.Bytes())
				})
			}
		}
	})
}

func (t *Encode) runStreamParallelCases(cases [][]encode.EncodedType) {
	t.Catch(func() {
		t.Run(fmt.Sprintf("stream-combination-%v-parallel", len(cases[0])), func() {
			data := t.randomData()
			wg := new(sync.WaitGroup)
			defer wg.Wait()
			for _, cs := range cases {
				for _, item := range t.generateStreamCases(cs...) {
					wg.Add(1)

					testCase := item
					go func() {
						defer wg.Done()

						codecStream := encode.NewCodecStream(testCase.extenders...)
						dataBuffer := bytes.NewReader(data)

						encodedBuffer := bytes.NewBuffer(nil)
						_, err := codecStream.Encode(encodedBuffer, dataBuffer)
						t.NoError(err)
						t.NotZero(encodedBuffer.Len())
						t.NotEqualValues(data, encodedBuffer.Bytes())

						decodedBuffer := bytes.NewBuffer(nil)
						_, err = codecStream.Decode(decodedBuffer, encodedBuffer)
						t.NoError(err)

						t.EqualValues(data, decodedBuffer.Bytes())
					}()
				}
			}
		})
	})
}

func (t *Encode) randomData() (data []byte) {
	const (
		jitterLength = 4 * 1024                   // 4kb
		largeLength  = 1024*1024 - jitterLength/2 // 1m - 2kb
	)

	// 1m ± 2kb
	data = make([]byte, largeLength+rand.Int()%(jitterLength/2))
	//data = make([]byte, 10)
	_, err := utils.Random(data, utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	return
}

func (t *Encode) smallRandomData() (data []byte) {
	const (
		jitterLength = 4 * 1024                 // 4kb
		smallLength  = 32*1024 - jitterLength/2 // 32kb - 2kb
	)

	// 32kb ± 2kb
	data = make([]byte, smallLength+rand.Int()%(jitterLength/2))
	_, err := utils.Random(data, utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	return
}

func (t *Encode) generateCodecCases(types ...encode.EncodedType) (options []*option) {
	opts := make(optionList, 0, len(types))
	for _, typ := range types {
		switch typ {
		case encode.EncodedTypeCipher:
			opts = append(opts, t.cipherOptions())
		case encode.EncodedTypeCompress:
			opts = append(opts, t.compressOptions())
		case encode.EncodedTypeEncode:
			opts = append(opts, t.printableOptions())
		}
	}

	return opts.combine()
}

func (t *Encode) generateStreamCases(types ...encode.EncodedType) (options []*option) {
	opts := make(optionList, 0, len(types))
	for _, typ := range types {
		switch typ {
		case encode.EncodedTypeCipher:
			opts = append(opts, t.cipherStreamOptions())
		case encode.EncodedTypeCompress:
			opts = append(opts, t.compressOptions())
		case encode.EncodedTypeEncode:
			opts = append(opts, t.printableOptions())
		}
	}

	return opts.combine()
}

func (t *Encode) cipherStreamOptions() (opt *option) {
	opt = new(option)

	cipherModes := []cipher.Mode{
		cipher.ModeCFB,
		cipher.ModeCTR,
		cipher.ModeOFB,
		cipher.ModeGCM,
	}

	// cipher Options
	var (
		k1        [1]byte
		k8, iv8   [8]byte
		k16, iv16 [16]byte
		k24       [24]byte
		k32       [32]byte
		k256      [256]byte
	)
	_, err := utils.Random(k1[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(k8[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(iv8[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(k16[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(iv16[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(k24[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(k32[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(k256[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)

	for _, mode := range cipherModes {
		if mode != cipher.ModeGCM {
			opt.names = append(opt.names, cipher.AlgorithmDES.String()+"-"+mode.String())
			opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmDES, mode, k8[:], iv8[:]))

			opt.names = append(opt.names, cipher.Algorithm3DES.String()+"-"+mode.String())
			opt.extenders = append(opt.extenders, encode.Cipher(cipher.Algorithm3DES, mode, k24[:], iv8[:]))
		}

		opt.names = append(opt.names, cipher.AlgorithmAES.String()+"-128-"+mode.String())
		opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmAES, mode, k16[:], iv16[:]))

		opt.names = append(opt.names, cipher.AlgorithmAES.String()+"-192-"+mode.String())
		opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmAES, mode, k24[:], iv16[:]))

		opt.names = append(opt.names, cipher.AlgorithmAES.String()+"-256-"+mode.String())
		opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmAES, mode, k32[:], iv16[:]))
	}

	opt.names = append(opt.names, cipher.AlgorithmRC4.String()+"-8")
	opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmRC4, 0, k1[:], nil))

	opt.names = append(opt.names, cipher.AlgorithmRC4.String()+"-256")
	opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmRC4, 0, k32[:], nil))

	opt.names = append(opt.names, cipher.AlgorithmRC4.String()+"-1024")
	opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmRC4, 0, k256[:], nil))

	opt.names = append(opt.names, cipher.AlgorithmChaCha20poly1305.String())
	opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmChaCha20poly1305, 0, k32[:], nil))

	opt.names = append(opt.names, cipher.AlgorithmXChaCha20poly1305.String())
	opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmXChaCha20poly1305, 0, k32[:], nil))

	return
}

func (t *Encode) cipherOptions() (opt *option) {
	opt = new(option)

	cipherModes := []cipher.Mode{
		cipher.ModeECB,
		cipher.ModeCBC,
		cipher.ModeCFB,
		cipher.ModeCTR,
		cipher.ModeOFB,
		cipher.ModeGCM,
	}

	// cipher Options
	var (
		k1        [1]byte
		k8, iv8   [8]byte
		k16, iv16 [16]byte
		k24       [24]byte
		k32       [32]byte
		k256      [256]byte
	)
	_, err := utils.Random(k1[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(k8[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(iv8[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(k16[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(iv16[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(k24[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(k32[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)
	_, err = utils.Random(k256[:], utils.GetTimeStamp(time.Now()))
	t.NoError(err)

	for _, mode := range cipherModes {
		if mode != cipher.ModeGCM {
			opt.names = append(opt.names, cipher.AlgorithmDES.String()+"-"+mode.String())
			opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmDES, mode, k8[:], iv8[:]))

			opt.names = append(opt.names, cipher.Algorithm3DES.String()+"-"+mode.String())
			opt.extenders = append(opt.extenders, encode.Cipher(cipher.Algorithm3DES, mode, k24[:], iv8[:]))
		}

		opt.names = append(opt.names, cipher.AlgorithmAES.String()+"-128-"+mode.String())
		opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmAES, mode, k16[:], iv16[:]))

		opt.names = append(opt.names, cipher.AlgorithmAES.String()+"-192-"+mode.String())
		opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmAES, mode, k24[:], iv16[:]))

		opt.names = append(opt.names, cipher.AlgorithmAES.String()+"-256-"+mode.String())
		opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmAES, mode, k32[:], iv16[:]))

		opt.names = append(opt.names, cipher.AlgorithmSM4.String()+"-"+mode.String())
		opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmSM4, mode, k16[:], iv16[:]))
	}

	opt.names = append(opt.names, cipher.AlgorithmRC4.String()+"-8")
	opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmRC4, 0, k1[:], nil))

	opt.names = append(opt.names, cipher.AlgorithmRC4.String()+"-256")
	opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmRC4, 0, k32[:], nil))

	opt.names = append(opt.names, cipher.AlgorithmRC4.String()+"-1024")
	opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmRC4, 0, k256[:], nil))

	opt.names = append(opt.names, cipher.AlgorithmChaCha20poly1305.String())
	opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmChaCha20poly1305, 0, k32[:], nil))

	opt.names = append(opt.names, cipher.AlgorithmXChaCha20poly1305.String())
	opt.extenders = append(opt.extenders, encode.Cipher(cipher.AlgorithmXChaCha20poly1305, 0, k32[:], nil))

	return
}

func (t *Encode) compressOptions() (opt *option) {
	opt = new(option)

	algos := []compress.Algorithm{
		compress.AlgorithmZSTD,
		compress.AlgorithmZLib,
		compress.AlgorithmS2,
		compress.AlgorithmGZip,
		compress.AlgorithmDeflate,
	}

	opt.names = make([]string, 0, len(algos))
	opt.extenders = make([]utils.OptionExtender, 0, len(algos))
	for _, algo := range algos {
		opt.names = append(opt.names, algo.String())
		opt.extenders = append(opt.extenders, encode.Compress(algo))
	}
	return
}

func (t *Encode) printableOptions() (opt *option) {
	opt = new(option)

	algos := []encode.Algorithm{
		encode.AlgorithmHex,
		encode.AlgorithmBase32Std,
		encode.AlgorithmBase32Hex,
		encode.AlgorithmBase64Std,
		encode.AlgorithmBase64URL,
		encode.AlgorithmBase64RawStd,
		encode.AlgorithmBase64RawURL,
	}

	opt.names = make([]string, 0, len(algos))
	opt.extenders = make([]utils.OptionExtender, 0, len(algos))
	for _, algo := range algos {
		opt.names = append(opt.names, algo.String())
		opt.extenders = append(opt.extenders, encode.Encode(algo))
	}
	return
}

type option struct {
	names     []string
	extenders []utils.OptionExtender
}

type optionList []*option

func (o optionList) combine() (result []*option) {
	type element struct {
		depth   int
		current *option
	}
	queue := []*element{
		{
			depth:   0,
			current: new(option),
		},
	}
	for len(queue) > 0 {
		elem := queue[0]
		queue = queue[1:]

		if elem.depth == len(o) {
			if len(elem.current.extenders) == len(o) {
				result = append(result, elem.current)
			}
			continue
		}

		opt := o[elem.depth]
		if len(opt.extenders) == 0 {
			queue = append(queue, &element{
				depth: elem.depth + 1,
				current: &option{
					names:     elem.current.names,
					extenders: elem.current.extenders,
				},
			})
			continue
		}

		for i := 0; i < len(opt.extenders); i++ {
			newElem := &element{
				depth: elem.depth + 1,
				current: &option{
					names:     append(elem.current.names, opt.names[i]),
					extenders: append(elem.current.extenders, opt.extenders[i]),
				},
			}
			queue = append(queue, newElem)
		}
	}

	return
}
