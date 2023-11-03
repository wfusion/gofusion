package encode

import (
	"hash/crc64"
	"math/rand"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wfusion/gofusion/common/utils/cipher"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/encode"

	"github.com/wfusion/gofusion/common/fus/debug"
)

var (
	opt    = new(option)
	orders []encode.EncodedType
)

type option struct {
	key, iv []byte

	cipherAlgo       cipher.Algorithm
	cipherAlgoString string

	cipherMode       cipher.Mode
	cipherModeString string

	compressAlgo       compress.Algorithm
	compressAlgoString string

	encodeAlgo       encode.Algorithm
	encodeAlgoString string

	output  string
	confuse bool

	orders []encode.EncodedType
}

type orderedStringFlag struct {
	t encode.EncodedType
	v *string
}

func (o *orderedStringFlag) String() string {
	return *o.v
}

func (o *orderedStringFlag) Set(val string) error {
	*o.v = val
	orders = append(orders, o.t)
	return nil
}

func (o *orderedStringFlag) Type() string {
	return "string"
}

func EncCommand() *cobra.Command {
	cmd := args(&cobra.Command{
		Use:     "enc [flags] [source]",
		Short:   "Encode data from stdin or filename",
		Example: "  fus enc -c aes -m cfb -k [base64] -iv [base64] -z gzip -e hex 'this is a plain text'",
		RunE:    runEnc,
	})

	return cmd
}

func DecCommand() *cobra.Command {
	cmd := args(&cobra.Command{
		Use:     "dec [flags] [source]",
		Short:   "Decode data from stdin or filename",
		Example: "  fus dec -c aes -m cfb -k [base64] -iv [base64] -z gzip -e hex 'this is an encoded text'",
		RunE:    runDec,
	})

	return cmd
}

func args(cmd *cobra.Command) *cobra.Command {
	cmd.Args = cobra.ExactArgs(1)

	// cipher
	cipherFlag := &orderedStringFlag{t: encode.EncodedTypeCipher, v: &opt.cipherAlgoString}
	cmd.Flags().VarP(cipherFlag, "cipher", "c",
		"cipher algorithm [des, 3des, aes, sm4, rc4, chacha20poly1305, xchacha20poly1305]")
	cmd.Flags().StringVarP(&opt.cipherModeString, "mode", "m", "",
		"cipher mode [ecb, cbc, cfb, ctr, ofb, gcm]")
	cmd.Flags().BytesBase64VarP(&opt.key, "key", "k", nil,
		"cipher key base64 format")
	cmd.Flags().BytesBase64VarP(&opt.iv, "iv", "i", nil,
		"cipher iv base64 format")

	// compress
	compressFlag := &orderedStringFlag{t: encode.EncodedTypeCompress, v: &opt.compressAlgoString}
	cmd.Flags().VarP(compressFlag, "compress", "z",
		"compress algorithm [zstd, zlib, s2, gzip, deflate]")

	// encode
	encodeFlag := &orderedStringFlag{t: encode.EncodedTypeEncode, v: &opt.encodeAlgoString}
	cmd.Flags().VarP(encodeFlag, "encode", "e",
		"printable encode algorithm [hex, base32, base32-hex, base64, base64-url, base64-raw, base64-raw-url]")

	// output
	cmd.Flags().StringVarP(&opt.output, "output", "o", "", "output to file")

	// confuse
	cmd.Flags().BoolVarP(&opt.confuse, "confuse-key", "", false, "confusing key")

	// check
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) (err error) {
		changed := false

		// check cipher algorithm
		if cmd.Flag("cipher").Changed {
			changed = true
			opt.cipherAlgo = cipher.ParseAlgorithm(opt.cipherAlgoString)
			if !opt.cipherAlgo.IsValid() {
				return errors.Errorf("unknown cipher algorihtm: %s.", opt.cipherAlgoString)
			}
			if !cmd.Flag("key").Changed {
				return errors.Errorf("cipher key flag unset.")
			}
			if opt.confuse {
				debug.Printf("confusing key\n")
				opt.key = confuseKey(opt.key)
			}
			if !cmd.Flag("mode").Changed &&
				opt.cipherAlgo != cipher.AlgorithmRC4 &&
				opt.cipherAlgo != cipher.AlgorithmChaCha20poly1305 &&
				opt.cipherAlgo != cipher.AlgorithmXChaCha20poly1305 {
				return errors.Errorf("cipher mode flag unset.")
			}
		}

		// check cipher mode
		if cmd.Flag("mode").Changed {
			if !cmd.Flag("cipher").Changed {
				return errors.Errorf("cipher algorithm flag unset.")
			}

			opt.cipherMode = cipher.ParseMode(opt.cipherModeString)
			if !opt.cipherMode.IsValid() {
				return errors.Errorf("unknown cipher mode: %s.", opt.cipherModeString)
			}
			if !cmd.Flag("iv").Changed &&
				!opt.cipherMode.SupportStream() {
				return errors.Errorf("cipher iv unset.")
			}
		}

		// check compress algorithm
		if cmd.Flag("compress").Changed {
			changed = true
			opt.compressAlgo = compress.ParseAlgorithm(opt.compressAlgoString)
			if !opt.compressAlgo.IsValid() {
				return errors.Errorf("unknown compress algorithm: %s.", opt.compressAlgoString)
			}
		}

		// check encode algorithm
		if cmd.Flag("encode").Changed {
			changed = true
			opt.encodeAlgo = encode.ParseAlgorithm(opt.encodeAlgoString)
			if !opt.encodeAlgo.IsValid() {
				return errors.Errorf("unknown encode algorithm: %s.", opt.encodeAlgoString)
			}
		}

		if !changed {
			return errors.Errorf("nothing to do.")
		}

		return
	}

	return cmd
}

func confuseKey(key []byte) (confused []byte) {
	var (
		k1 = make([]byte, len(key))
		k2 = make([]byte, len(key))
		k3 = make([]byte, len(key))
	)
	rndSeed := int64(crc64.Checksum(key, crc64.MakeTable(crc64.ISO)))
	rand.New(rand.NewSource(cipher.RndSeed ^ compress.RndSeed ^ rndSeed)).Read(k1)
	rand.New(rand.NewSource(cipher.RndSeed ^ encode.RndSeed ^ rndSeed)).Read(k2)
	rand.New(rand.NewSource(compress.RndSeed ^ encode.RndSeed ^ rndSeed)).Read(k3)

	confused = make([]byte, len(key))
	rand.New(rand.NewSource(cipher.RndSeed ^ compress.RndSeed ^ encode.RndSeed)).Read(confused)
	for i := 0; i < len(confused); i++ {
		confused[i] ^= k1[i] ^ k2[i] ^ k3[i]
	}
	return
}
