package cases

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/des"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/chacha20poly1305"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/cipher"
	"github.com/wfusion/gofusion/log"

	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestCipher(t *testing.T) {
	t.Parallel()
	testingSuite := &Cipher{Test: new(testUtl.Test)}
	suite.Run(t, testingSuite)
}

type Cipher struct {
	*testUtl.Test
}

func (t *Cipher) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Cipher) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Cipher) TestDES() {
	t.Catch(func() {
		var (
			key [8]byte
			iv  [des.BlockSize]byte
		)
		_, err := utils.Random(key[:], 0)
		t.NoError(err)
		_, err = utils.Random(iv[:], 0)
		t.NoError(err)

		t.runTest(cipher.AlgorithmDES, key[:], iv[:], cipher.ModeGCM)
	})
}

func (t *Cipher) Test3DES() {
	t.Catch(func() {
		var (
			key [8 * 3]byte
			iv  [des.BlockSize]byte
		)
		_, err := utils.Random(key[:], 0)
		t.NoError(err)
		_, err = utils.Random(iv[:], 0)
		t.NoError(err)

		t.runTest(cipher.Algorithm3DES, key[:], iv[:], cipher.ModeGCM)
	})
}

func (t *Cipher) TestAES128() {
	t.Catch(func() {
		var (
			key [16]byte
			iv  [aes.BlockSize]byte
		)
		_, err := utils.Random(key[:], 0)
		t.NoError(err)
		_, err = utils.Random(iv[:], 0)
		t.NoError(err)

		t.runTest(cipher.AlgorithmAES, key[:], iv[:])
	})
}

func (t *Cipher) TestAES192() {
	t.Catch(func() {
		var (
			key [24]byte
			iv  [aes.BlockSize]byte
		)
		_, err := utils.Random(key[:], 0)
		t.NoError(err)
		_, err = utils.Random(iv[:], 0)
		t.NoError(err)

		t.runTest(cipher.AlgorithmAES, key[:], iv[:])
	})
}

func (t *Cipher) TestAES256() {
	t.Catch(func() {
		var (
			key [32]byte
			iv  [aes.BlockSize]byte
		)
		_, err := utils.Random(key[:], 0)
		t.NoError(err)
		_, err = utils.Random(iv[:], 0)
		t.NoError(err)

		t.runTest(cipher.AlgorithmAES, key[:], iv[:])
	})
}

func (t *Cipher) TestRC4_8() {
	t.Catch(func() {
		var (
			key [1]byte
		)
		_, err := utils.Random(key[:], 0)
		t.NoError(err)

		t.runTest(cipher.AlgorithmRC4, key[:], nil,
			cipher.ModeCBC, cipher.ModeCFB, cipher.ModeCTR, cipher.ModeOFB, cipher.ModeGCM)
	})
}

func (t *Cipher) TestRC4_256() {
	t.Catch(func() {
		var (
			key [32]byte
		)
		_, err := utils.Random(key[:], 0)
		t.NoError(err)

		t.runTest(cipher.AlgorithmRC4, key[:], nil,
			cipher.ModeCBC, cipher.ModeCFB, cipher.ModeCTR, cipher.ModeOFB, cipher.ModeGCM)
	})
}

func (t *Cipher) TestRC4_2048() {
	t.Catch(func() {
		var (
			key [256]byte
		)
		_, err := utils.Random(key[:], 0)
		t.NoError(err)

		t.runTest(cipher.AlgorithmRC4, key[:], nil,
			cipher.ModeCBC, cipher.ModeCFB, cipher.ModeCTR, cipher.ModeOFB, cipher.ModeGCM)
	})
}

func (t *Cipher) TestChaCha20poly1305() {
	t.Catch(func() {
		var (
			key [chacha20poly1305.KeySize]byte
		)
		_, err := utils.Random(key[:], 0)
		t.NoError(err)

		t.runTest(cipher.AlgorithmChaCha20poly1305, key[:], nil,
			cipher.ModeCBC, cipher.ModeCFB, cipher.ModeCTR, cipher.ModeOFB, cipher.ModeGCM)
	})
}

func (t *Cipher) TestXChaCha20poly1305() {
	t.Catch(func() {
		var (
			key [chacha20poly1305.KeySize]byte
		)
		_, err := utils.Random(key[:], 0)
		t.NoError(err)

		t.runTest(cipher.AlgorithmXChaCha20poly1305, key[:], nil,
			cipher.ModeCBC, cipher.ModeCFB, cipher.ModeCTR, cipher.ModeOFB, cipher.ModeGCM)
	})
}

func (t *Cipher) TestSM4() {
	t.Catch(func() {
		var (
			key [16]byte
			iv  [aes.BlockSize]byte
		)
		_, err := utils.Random(key[:], 0)
		t.NoError(err)
		_, err = utils.Random(iv[:], 0)
		t.NoError(err)

		t.runTest(cipher.AlgorithmSM4, key[:], iv[:])
	})
}

func (t *Cipher) runTest(algo cipher.Algorithm, key, iv []byte, ignoreModes ...cipher.Mode) {
	testCases := []testCipherFunc{
		t.testModes,
		t.testLargeBytes,
		t.testBytesParallel,
		t.testStreaming,
		t.testStreamingParallel,
	}

	rand.Seed(utils.GetTimeStamp(time.Now()))
	for _, testCase := range testCases {
		testCase(algo, key, iv, ignoreModes...)
	}
}

type testCipherFunc func(algo cipher.Algorithm, key, iv []byte, ignoreModes ...cipher.Mode)

func (t *Cipher) testModes(algo cipher.Algorithm, key, iv []byte, ignoreModes ...cipher.Mode) {
	type caseStruct struct {
		data []byte
		mode cipher.Mode
	}

	caseList := []caseStruct{
		{
			data: []byte("this is a plain text."),
			mode: cipher.ModeECB,
		},
		{
			data: []byte("this is a plain text."),
			mode: cipher.ModeCBC,
		},
		{
			data: []byte("this is a plain text."),
			mode: cipher.ModeCFB,
		},
		{
			data: []byte("this is a plain text."),
			mode: cipher.ModeCTR,
		},
		{
			data: []byte("this is a plain text."),
			mode: cipher.ModeOFB,
		},
		{
			data: []byte("this is a plain text."),
			mode: cipher.ModeGCM,
		},
	}

	ignored := utils.NewSet(ignoreModes...)
	for _, cs := range caseList {
		if ignored.Contains(cs.mode) {
			continue
		}

		name := algo.String() + "_" + cs.mode.String()
		t.Run(name, func() {
			enc, err := cipher.EncryptBytesFunc(algo, cs.mode, key, iv)
			t.NoError(err)
			dec, err := cipher.DecryptBytesFunc(algo, cs.mode, key, iv)
			t.NoError(err)

			ciphertext, err := enc(cs.data)
			t.NoError(err)
			t.NotEmpty(ciphertext)
			t.NotEqualValues(cs.data, ciphertext)

			actual, err := dec(ciphertext)
			t.NoError(err)

			t.EqualValues(cs.data, actual)
		})
	}
}

func (t *Cipher) testLargeBytes(algo cipher.Algorithm, key, iv []byte, ignoreModes ...cipher.Mode) {
	type caseStruct struct {
		mode cipher.Mode
	}

	caseList := []caseStruct{
		{
			mode: cipher.ModeECB,
		},
		{
			mode: cipher.ModeCBC,
		},
		{
			mode: cipher.ModeCFB,
		},
		{
			mode: cipher.ModeCTR,
		},
		{
			mode: cipher.ModeOFB,
		},
		{
			mode: cipher.ModeGCM,
		},
	}

	ignored := utils.NewSet(ignoreModes...)
	data := t.randomData()
	for _, cs := range caseList {
		if ignored.Contains(cs.mode) {
			continue
		}
		name := algo.String() + "_" + cs.mode.String() + "_large_bytes"
		t.Run(name, func() {
			enc, err := cipher.EncryptBytesFunc(algo, cs.mode, key, iv)
			t.NoError(err)
			dec, err := cipher.DecryptBytesFunc(algo, cs.mode, key, iv)
			t.NoError(err)

			ciphertext, err := enc(data)
			t.NoError(err)
			t.NotEmpty(ciphertext)
			t.NotEqualValues(data, ciphertext)

			actual, err := dec(ciphertext)
			t.NoError(err)

			t.EqualValues(data, actual)
		})
	}
}

func (t *Cipher) testBytesParallel(algo cipher.Algorithm, key, iv []byte, ignoreModes ...cipher.Mode) {
	type caseStruct struct {
		mode cipher.Mode
	}

	caseList := []caseStruct{
		{
			mode: cipher.ModeECB,
		},
		{
			mode: cipher.ModeCBC,
		},
		{
			mode: cipher.ModeCFB,
		},
		{
			mode: cipher.ModeCTR,
		},
		{
			mode: cipher.ModeOFB,
		},
		{
			mode: cipher.ModeGCM,
		},
	}

	ignored := utils.NewSet(ignoreModes...)
	name := algo.String() + "_bytes_parallel"
	t.Run(name, func() {
		wg := new(sync.WaitGroup)
		defer wg.Wait()

		data := t.randomData()
		for _, cs := range caseList {
			if ignored.Contains(cs.mode) {
				continue
			}

			mode := cs.mode
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					enc, err := cipher.EncryptBytesFunc(algo, mode, key, iv)
					t.NoError(err)
					dec, err := cipher.DecryptBytesFunc(algo, mode, key, iv)
					t.NoError(err)

					ciphertext, err := enc(data)
					t.NoError(err)
					t.NotEmpty(ciphertext)
					t.NotEqualValues(data, ciphertext)

					actual, err := dec(ciphertext)
					t.NoError(err)

					t.EqualValues(data, actual)
				}()
			}
		}
	})
}

func (t *Cipher) testStreaming(algo cipher.Algorithm, key, iv []byte, ignoreModes ...cipher.Mode) {
	type caseStruct struct {
		mode cipher.Mode
	}

	caseList := []caseStruct{
		{
			mode: cipher.ModeECB,
		},
		{
			mode: cipher.ModeCBC,
		},
		{
			mode: cipher.ModeCFB,
		},
		{
			mode: cipher.ModeCTR,
		},
		{
			mode: cipher.ModeOFB,
		},
		{
			mode: cipher.ModeGCM,
		},
	}

	ignored := utils.NewSet(ignoreModes...)
	data := t.randomData()
	for _, cs := range caseList {
		if ignored.Contains(cs.mode) {
			continue
		}
		if _, err := cipher.EncryptStreamFunc(algo, cs.mode, key, iv); errors.Is(err, cipher.ErrNotSupportStream) {
			continue
		}

		mode := cs.mode
		name := algo.String() + "_" + mode.String() + "_streaming"
		t.Run(name, func() {
			dataBuffer := bytes.NewReader(data)

			enc, err := cipher.EncryptStreamFunc(algo, mode, key, iv)
			t.NoError(err)
			dec, err := cipher.DecryptStreamFunc(algo, mode, key, iv)
			t.NoError(err)

			cipherBuffer := bytes.NewBuffer(nil)
			err = enc(cipherBuffer, dataBuffer)
			t.NoError(err)
			t.NotZero(cipherBuffer.Len())
			t.NotEqualValues(data, cipherBuffer.Bytes())

			plainBuffer := bytes.NewBuffer(nil)
			err = dec(plainBuffer, cipherBuffer)
			t.NoError(err)

			t.EqualValues(data, plainBuffer.Bytes())
		})
	}
}

func (t *Cipher) testStreamingParallel(algo cipher.Algorithm, key, iv []byte, ignoreModes ...cipher.Mode) {
	type caseStruct struct {
		mode cipher.Mode
	}

	caseList := []caseStruct{
		{
			mode: cipher.ModeECB,
		},
		{
			mode: cipher.ModeCBC,
		},
		{
			mode: cipher.ModeCFB,
		},
		{
			mode: cipher.ModeCTR,
		},
		{
			mode: cipher.ModeOFB,
		},
		{
			mode: cipher.ModeGCM,
		},
	}

	ignored := utils.NewSet(ignoreModes...)
	name := algo.String() + "_stream_parallel"
	t.Run(name, func() {
		wg := new(sync.WaitGroup)
		defer wg.Wait()

		data := t.randomData()
		for _, cs := range caseList {
			if ignored.Contains(cs.mode) {
				continue
			}
			if _, err := cipher.EncryptStreamFunc(algo, cs.mode, key, iv); errors.Is(err, cipher.ErrNotSupportStream) {
				continue
			}

			mode := cs.mode
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					dataBuffer := bytes.NewReader(data)

					enc, err := cipher.EncryptStreamFunc(algo, mode, key, iv)
					t.NoError(err)
					dec, err := cipher.DecryptStreamFunc(algo, mode, key, iv)
					t.NoError(err)

					cipherBuffer := bytes.NewBuffer(nil)
					err = enc(cipherBuffer, dataBuffer)
					t.NoError(err)
					t.NotZero(cipherBuffer.Len())
					t.NotEqualValues(data, cipherBuffer.Bytes())

					plainBuffer := bytes.NewBuffer(nil)
					err = dec(plainBuffer, cipherBuffer)
					t.NoError(err)

					t.EqualValues(data, plainBuffer.Bytes())
				}()
			}
		}
	})

}

func (t *Cipher) randomData() (data []byte) {
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
