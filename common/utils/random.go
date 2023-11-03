package utils

import (
	cryptoRand "crypto/rand"
	"encoding/ascii85"
	"encoding/base64"
	"encoding/hex"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lithammer/shortuuid/v4"
	"github.com/oklog/ulid/v2"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils/inspect"
)

func UUID() string {
	return uuid.New().String()
}

// UUID_
//nolint: revive // uuid without hyphen function issue
func UUID_() string {
	return strings.Replace(uuid.New().String(), constant.Hyphen, "", -1)
}

// ShortUUID returns a new short UUID with base57
func ShortUUID() string {
	return shortuuid.New()
}

// ULID returns a new ULID.
func ULID() string {
	return ulid.MustNew(ulid.Now(), cryptoRand.Reader).String()
}

func UUID20() string {
	id := uuid.New()
	t := make([]byte, ascii85.MaxEncodedLen(len(id)))
	n := ascii85.Encode(t, id[:])
	return string(t[:n])
}

func UUID22() string {
	id := uuid.New()
	return base64.RawURLEncoding.EncodeToString(id[:])
}

const (
	randomChars       = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	randomCharsLength = len(randomChars)
)

func CryptoRandom(b []byte) (n int, err error) {
	return cryptoRand.Read(b)
}

func CryptoRandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	if _, err := cryptoRand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func CryptoRandomLetterAndNumber(n int) (string, error) {
	b, err := CryptoRandomBytes(n)
	if err != nil {
		return "", err
	}

	random := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		random = append(random, randomChars[b[i]%uint8(randomCharsLength)])
	}
	return string(random), nil
}

func RandomLetterAndNumber(n int) string {
	random := make([]byte, 0, n)
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < n; i++ {
		random = append(random, randomChars[rand.Intn(randomCharsLength)])
	}

	return string(random)
}

func Random(b []byte, seed int64) (n int, err error) {
	if seed == 0 {
		rand.Seed(time.Now().UnixNano())
	} else {
		rand.Seed(seed)
	}
	return rand.Read(b)
}

func RandomNumbers(n int) string {
	rand.Seed(time.Now().UnixNano())
	ret := ""
	for i := 0; i < n; i++ {
		ret += strconv.Itoa(rand.Intn(10))
	}
	return ret
}

func CryptoRandomNumbers(n int) (string, error) {
	b, err := CryptoRandomBytes(n)
	if err != nil {
		return "", err
	}

	random := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		random = append(random, randomChars[b[i]%uint8(10)])
	}
	return string(random), nil
}

func NginxID() string {
	upper := func(c byte) byte {
		val := c
		if val >= 97 && val <= 122 {
			return val - 32
		}
		return c
	}
	int2byte := func(bs []byte, val int) {
		size := 10
		l := len(bs) - 1
		for idx := l; idx >= 0; idx-- {
			bs[idx] = byte(uint(val%size) + uint('0'))
			val = val / size
		}
	}

	ret := [33]byte{}
	t := time.Now()
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	int2byte(ret[:4], year)
	int2byte(ret[4:6], int(month))
	int2byte(ret[6:8], day)
	int2byte(ret[8:10], hour)
	int2byte(ret[10:12], minute)
	int2byte(ret[12:14], second)
	copy(ret[14:26], LocalIP.Bytes())
	ms := t.UnixNano() / 1e6 % 1000
	int2byte(ret[26:29], int(ms))
	u32 := rand.Uint32()
	u32 >>= 16
	src := []byte{byte(u32 & 0xff), byte((u32 >> 8) & 0xff)}
	hex.Encode(ret[29:33], src)
	for idx := 29; idx < 33; idx++ {
		ret[idx] = upper(ret[idx])
	}
	return string(ret[:])
}

func NewSafeRand(seed int64) *rand.Rand {
	return rand.New(newBuiltinLockedSource(seed))
}

func newBuiltinLockedSource(seed int64) rand.Source64 {
	t := inspect.TypeOf("math/rand.lockedSource")
	source := reflect.New(t).Interface()

	var err error
	IfAny(
		// go1.16 - go1.19
		func() bool {
			_, err = Catch(func() { inspect.SetField(source, "src", rand.NewSource(seed)) })
			return err == nil
		},
		// go1.20 - go1.21, source is renamed and set seed when calling rather than constructing stage
		func() bool {
			_, err = Catch(func() { inspect.SetField(source, "s", rand.NewSource(seed)) })
			return err == nil
		},
	)
	if err != nil {
		panic(err)
	}

	return source.(rand.Source64)
}

func init() {
	rand.Seed(time.Now().Unix())

	// assert if rand.lockedSource struct is not changed
	newBuiltinLockedSource(1).Int63()
}
