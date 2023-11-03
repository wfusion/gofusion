package mock

import (
	"math"
	"time"
	"unsafe"

	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/interfaces"
	"github.com/go-faker/faker/v4/pkg/options"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize"
)

type randomOption struct {
	ignored []string
}

func IgnoreFields(ignored ...string) utils.OptionFunc[randomOption] {
	return func(o *randomOption) {
		o.ignored = ignored
	}
}

func GenObjBySerializeAlgo(algo serialize.Algorithm) (data any) {
	switch algo {
	case serialize.AlgorithmGob:
		// gob may unmarshal time.Time value to a different internal member time.Time
		// gob unmarshal false boolean pointer as a nil value
		ignoreFields := []string{"Time", "Timep", "Boolp"}
		data = newRandomObj(IgnoreFields(ignoreFields...))
	case serialize.AlgorithmJson, serialize.AlgorithmMsgpack, serialize.AlgorithmCbor:
		// json unmarshal all number to any type as float64 or json.Number type
		// msgpack unmarshal a compact number type to any type, e.g. int -> int8
		// cbor unmarshal integer type to any type as uint64
		ignoreFields := []string{"AnySlice", "AnypSlice"}
		data = newCommonObj(IgnoreFields(ignoreFields...))
	default:
		data = newRandomObj()
	}
	return
}

func GenObjListBySerializeAlgo(algo serialize.Algorithm, num int) (dataList any) {
	switch algo {
	case serialize.AlgorithmGob:
		// gob may unmarshal time.Time value to a different internal member time.Time
		// gob unmarshal false boolean pointer as a nil value
		ignoreFields := []string{"Time", "Timep", "Boolp"}
		dataList = newRandomObjList(num, IgnoreFields(ignoreFields...))
	case serialize.AlgorithmJson, serialize.AlgorithmMsgpack, serialize.AlgorithmCbor:
		// json unmarshal all number to any type as float64 or json.Number type
		// msgpack unmarshal a compact number type to any type, e.g. int -> int8
		// cbor unmarshal integer type to any type as uint64
		ignoreFields := []string{"AnySlice", "AnypSlice"}
		dataList = newCommonObjList(num, IgnoreFields(ignoreFields...))
	default:
		dataList = newRandomObjList(num)
	}
	return
}

func newCommonObj(opts ...utils.OptionExtender) (v *CommonObj) {
	opt := utils.ApplyOptions[randomOption](opts...)
	v = &CommonObj{
		CommonType: *mockCommonType(opt),
		Basic:      *mockCommonType(opt),
		Basicp:     mockCommonType(opt),
		Int64Map: map[int64]*CommonType{
			-1: mockCommonType(opt),
			0:  mockCommonType(opt),
			1:  mockCommonType(opt),
			2:  mockCommonType(opt),
		},
		StringMap: map[string]*CommonType{
			"":       mockCommonType(opt),
			"string": mockCommonType(opt),
		},
		Array: [2]*CommonType{mockCommonType(opt), mockCommonType(opt)},
	}

	return
}

func newCommonObjList(num int, opts ...utils.OptionExtender) (vList []*CommonObj) {
	vList = make([]*CommonObj, 0, num)
	for i := 0; i < num; i++ {
		vList = append(vList, newCommonObj(opts...))
	}
	return
}

type CommonObj struct {
	CommonType

	Basic  CommonType
	Basicp *CommonType

	Int64Map  map[int64]*CommonType
	StringMap map[string]*CommonType

	Array [2]*CommonType

	Nil *CommonType
}

func (r *CommonObj) EventType() string {
	return "common_object_created"
}

type CommonType struct {
	Str  string
	Strp *string

	StrSlice  []string
	StrpSlice []*string

	Rune  rune
	Runep *rune

	RuneSlice  []rune
	RunepSlice []*rune

	Byte  byte
	Bytep *byte

	ByteSlice  []byte
	BytepSlice []*byte

	Bool  bool
	Boolp *bool

	BoolSlice  []bool
	BoolpSlice []*bool

	Int   int
	Int8  int8
	Int16 int16
	Int32 int32
	Int64 int64

	IntSlice   []int
	Int8Slice  []int8
	Int16Slice []int16
	Int32Slice []int32
	Int64Slice []int64

	Intp   *int
	Int8p  *int8
	Int16p *int16
	Int32p *int32
	Int64p *int64

	IntpSlice   []*int
	Int8pSlice  []*int8
	Int16pSlice []*int16
	Int32pSlice []*int32
	Int64pSlice []*int64

	Uint   uint
	Uint8  uint8
	Uint16 uint16
	Uint32 uint32
	Uint64 uint64

	UintSlice   []uint
	Uint8Slice  []uint8
	Uint16Slice []uint16
	Uint32Slice []uint32
	Uint64Slice []uint64

	Uintp   *uint
	Uint8p  *uint8
	Uint16p *uint16
	Uint32p *uint32
	Uint64p *uint64

	UintpSlice   []*uint
	Uint8pSlice  []*uint8
	Uint16pSlice []*uint16
	Uint32pSlice []*uint32
	Uint64pSlice []*uint64

	Float32 float32
	Float64 float64

	Float32Slice []float32
	Float64Slice []float64

	Float32p *float32
	Float64p *float64

	Float32pSlice []*float32
	Float64pSlice []*float64

	Any  any  `faker:"-"`
	Anyp *any `faker:"-"`

	AnySlice  []any  `faker:"-"`
	AnypSlice []*any `faker:"-"`
}

func mockCommonType(opt *randomOption) (v *CommonType) {
	utils.MustSuccess(faker.FakeData(&v,
		options.WithNilIfLenIsZero(true),
		options.WithGenerateUniqueValues(true),
		options.WithRandomIntegerBoundaries(interfaces.RandomIntegerBoundary{Start: 1, End: math.MaxInt8}),
		options.WithRandomFloatBoundaries(interfaces.RandomFloatBoundary{Start: 1, End: math.MaxInt8}),
		options.WithFieldsToIgnore(opt.ignored...),
	))

	ignored := utils.NewSet(opt.ignored...)
	setter := func(name string, fn func()) {
		if !ignored.Contains(name) {
			fn()
		}
	}

	setter("Strp", func() { v.Strp = utils.AnyPtr(v.Str) })
	setter("StrpSlice", func() { v.StrpSlice = utils.SliceMapping(v.StrSlice, utils.AnyPtr[string]) })

	setter("Runep", func() { v.Runep = utils.AnyPtr(v.Rune) })
	setter("RunepSlice", func() { v.RunepSlice = utils.SliceMapping(v.RuneSlice, utils.AnyPtr[rune]) })

	setter("Bytep", func() { v.Bytep = utils.AnyPtr(v.Byte) })
	setter("BytepSlice", func() { v.BytepSlice = utils.SliceMapping(v.ByteSlice, utils.AnyPtr[byte]) })

	setter("Boolp", func() { v.Boolp = utils.AnyPtr(v.Bool) })

	setter("Intp", func() { v.Intp = utils.AnyPtr(v.Int) })
	setter("Int8p", func() { v.Int8p = utils.AnyPtr(v.Int8) })
	setter("Int16p", func() { v.Int16p = utils.AnyPtr(v.Int16) })
	setter("Int32p", func() { v.Int32p = utils.AnyPtr(v.Int32) })
	setter("Int64p", func() { v.Int64p = utils.AnyPtr(v.Int64) })

	setter("IntpSlice", func() { v.IntpSlice = utils.SliceMapping(v.IntSlice, utils.AnyPtr[int]) })
	setter("Int8pSlice", func() { v.Int8pSlice = utils.SliceMapping(v.Int8Slice, utils.AnyPtr[int8]) })
	setter("Int16pSlice", func() { v.Int16pSlice = utils.SliceMapping(v.Int16Slice, utils.AnyPtr[int16]) })
	setter("Int32pSlice", func() { v.Int32pSlice = utils.SliceMapping(v.Int32Slice, utils.AnyPtr[int32]) })
	setter("Int64pSlice", func() { v.Int64pSlice = utils.SliceMapping(v.Int64Slice, utils.AnyPtr[int64]) })

	setter("Uintp", func() { v.Uintp = utils.AnyPtr(v.Uint) })
	setter("Uint8p", func() { v.Uint8p = utils.AnyPtr(v.Uint8) })
	setter("Uint16p", func() { v.Uint16p = utils.AnyPtr(v.Uint16) })
	setter("Uint32p", func() { v.Uint32p = utils.AnyPtr(v.Uint32) })
	setter("Uint64p", func() { v.Uint64p = utils.AnyPtr(v.Uint64) })
	setter("UintpSlice", func() { v.UintpSlice = utils.SliceMapping(v.UintSlice, utils.AnyPtr[uint]) })
	setter("Uint8pSlice", func() { v.Uint8pSlice = utils.SliceMapping(v.Uint8Slice, utils.AnyPtr[uint8]) })
	setter("Uint16pSlice", func() { v.Uint16pSlice = utils.SliceMapping(v.Uint16Slice, utils.AnyPtr[uint16]) })
	setter("Uint32pSlice", func() { v.Uint32pSlice = utils.SliceMapping(v.Uint32Slice, utils.AnyPtr[uint32]) })
	setter("Uint64pSlice", func() { v.Uint64pSlice = utils.SliceMapping(v.Uint64Slice, utils.AnyPtr[uint64]) })

	setter("Float32p", func() { v.Float32p = utils.AnyPtr(v.Float32) })
	setter("Float64p", func() { v.Float64p = utils.AnyPtr(v.Float64) })
	setter("Float32pSlice", func() { v.Float32pSlice = utils.SliceMapping(v.Float32Slice, utils.AnyPtr[float32]) })
	setter("Float64pSlice", func() { v.Float64pSlice = utils.SliceMapping(v.Float64Slice, utils.AnyPtr[float64]) })

	setter("Any", func() { v.Any = "any" })
	setter("Anyp", func() { v.Anyp = utils.AnyPtr(v.Any) })
	setter("AnySlice", func() { v.AnySlice = []any{1, 1.1, "any", true, nil} })
	setter("AnypSlice", func() { v.AnypSlice = utils.SliceMapping(v.AnySlice, utils.AnyPtr[any]) })

	return
}

func newRandomObj(opts ...utils.OptionExtender) (v *RandomObj) {
	opt := utils.ApplyOptions[randomOption](opts...)
	v = &RandomObj{
		AllType: *mockAllType(opt),
		Basic:   *mockAllType(opt),
		Basicp:  mockAllType(opt),
		Int64Map: map[int64]*AllType{
			-1: mockAllType(opt),
			0:  mockAllType(opt),
			1:  mockAllType(opt),
			2:  mockAllType(opt),
		},
		Float64Map: map[float64]*AllType{
			-1.1: mockAllType(opt),
			-0.0: mockAllType(opt),
			0.1:  mockAllType(opt),
			1.1:  mockAllType(opt),
		},
		StringMap: map[string]*AllType{
			"":       mockAllType(opt),
			"string": mockAllType(opt),
		},
		Array: [2]*AllType{mockAllType(opt), mockAllType(opt)},
	}

	return
}

func newRandomObjList(num int, opts ...utils.OptionExtender) (vList []*RandomObj) {
	vList = make([]*RandomObj, 0, num)
	for i := 0; i < num; i++ {
		vList = append(vList, newRandomObj(opts...))
	}
	return
}

type RandomObj struct {
	AllType

	Basic  AllType
	Basicp *AllType

	Int64Map   map[int64]*AllType
	Float64Map map[float64]*AllType
	StringMap  map[string]*AllType

	Array [2]*AllType

	Nil *AllType
}

func (r *RandomObj) EventType() string {
	return "random_object_created"
}

type AllType struct {
	Str  string
	Strp *string

	StrSlice  []string
	StrpSlice []*string

	Rune  rune
	Runep *rune

	RuneSlice  []rune
	RunepSlice []*rune

	Byte  byte
	Bytep *byte

	ByteSlice  []byte
	BytepSlice []*byte

	Bool  bool
	Boolp *bool

	BoolSlice  []bool
	BoolpSlice []*bool

	Int   int
	Int8  int8
	Int16 int16
	Int32 int32
	Int64 int64

	IntSlice   []int
	Int8Slice  []int8
	Int16Slice []int16
	Int32Slice []int32
	Int64Slice []int64

	Intp   *int
	Int8p  *int8
	Int16p *int16
	Int32p *int32
	Int64p *int64

	IntpSlice   []*int
	Int8pSlice  []*int8
	Int16pSlice []*int16
	Int32pSlice []*int32
	Int64pSlice []*int64

	Uint   uint
	Uint8  uint8
	Uint16 uint16
	Uint32 uint32
	Uint64 uint64

	UintSlice   []uint
	Uint8Slice  []uint8
	Uint16Slice []uint16
	Uint32Slice []uint32
	Uint64Slice []uint64

	Uintp   *uint
	Uint8p  *uint8
	Uint16p *uint16
	Uint32p *uint32
	Uint64p *uint64

	UintpSlice   []*uint
	Uint8pSlice  []*uint8
	Uint16pSlice []*uint16
	Uint32pSlice []*uint32
	Uint64pSlice []*uint64

	Float32 float32
	Float64 float64

	Float32Slice []float32
	Float64Slice []float64

	Float32p *float32
	Float64p *float64

	Float32pSlice []*float32
	Float64pSlice []*float64

	Complex64  complex64  `faker:"-" json:"-"`
	Complex128 complex128 `faker:"-" json:"-"`

	Complex64Slice  []complex64  `faker:"-" json:"-"`
	Complex128Slice []complex128 `faker:"-" json:"-"`

	Complex64p  *complex64  `faker:"-" json:"-"`
	Complex128p *complex128 `faker:"-" json:"-"`

	Complex64pSlice  []*complex64  `faker:"-" json:"-"`
	Complex128pSlice []*complex128 `faker:"-" json:"-"`

	Uintptr  uintptr  `faker:"-"`
	Uintptrp *uintptr `faker:"-"`

	UintptrSlice  []uintptr  `faker:"-"`
	UintptrpSlice []*uintptr `faker:"-"`

	Any  any  `faker:"-"`
	Anyp *any `faker:"-"`

	AnySlice  []any  `faker:"-"`
	AnypSlice []*any `faker:"-"`

	Time  time.Time
	Timep *time.Time
}

func mockAllType(opt *randomOption) (v *AllType) {
	utils.MustSuccess(faker.FakeData(&v,
		options.WithNilIfLenIsZero(true),
		options.WithGenerateUniqueValues(true),
		options.WithRandomIntegerBoundaries(interfaces.RandomIntegerBoundary{Start: 1, End: math.MaxInt8}),
		options.WithRandomFloatBoundaries(interfaces.RandomFloatBoundary{Start: 1, End: math.MaxInt8}),
		options.WithFieldsToIgnore(opt.ignored...),
	))

	ignored := utils.NewSet(opt.ignored...)
	setter := func(name string, fn func()) {
		if !ignored.Contains(name) {
			fn()
		}
	}

	setter("Strp", func() { v.Strp = utils.AnyPtr(v.Str) })
	setter("StrpSlice", func() { v.StrpSlice = utils.SliceMapping(v.StrSlice, utils.AnyPtr[string]) })

	setter("Runep", func() { v.Runep = utils.AnyPtr(v.Rune) })
	setter("RunepSlice", func() { v.RunepSlice = utils.SliceMapping(v.RuneSlice, utils.AnyPtr[rune]) })

	setter("Bytep", func() { v.Bytep = utils.AnyPtr(v.Byte) })
	setter("BytepSlice", func() { v.BytepSlice = utils.SliceMapping(v.ByteSlice, utils.AnyPtr[byte]) })

	setter("Boolp", func() { v.Boolp = utils.AnyPtr(v.Bool) })

	setter("Intp", func() { v.Intp = utils.AnyPtr(v.Int) })
	setter("Int8p", func() { v.Int8p = utils.AnyPtr(v.Int8) })
	setter("Int16p", func() { v.Int16p = utils.AnyPtr(v.Int16) })
	setter("Int32p", func() { v.Int32p = utils.AnyPtr(v.Int32) })
	setter("Int64p", func() { v.Int64p = utils.AnyPtr(v.Int64) })

	setter("IntpSlice", func() { v.IntpSlice = utils.SliceMapping(v.IntSlice, utils.AnyPtr[int]) })
	setter("Int8pSlice", func() { v.Int8pSlice = utils.SliceMapping(v.Int8Slice, utils.AnyPtr[int8]) })
	setter("Int16pSlice", func() { v.Int16pSlice = utils.SliceMapping(v.Int16Slice, utils.AnyPtr[int16]) })
	setter("Int32pSlice", func() { v.Int32pSlice = utils.SliceMapping(v.Int32Slice, utils.AnyPtr[int32]) })
	setter("Int64pSlice", func() { v.Int64pSlice = utils.SliceMapping(v.Int64Slice, utils.AnyPtr[int64]) })

	setter("Uintp", func() { v.Uintp = utils.AnyPtr(v.Uint) })
	setter("Uint8p", func() { v.Uint8p = utils.AnyPtr(v.Uint8) })
	setter("Uint16p", func() { v.Uint16p = utils.AnyPtr(v.Uint16) })
	setter("Uint32p", func() { v.Uint32p = utils.AnyPtr(v.Uint32) })
	setter("Uint64p", func() { v.Uint64p = utils.AnyPtr(v.Uint64) })
	setter("UintpSlice", func() { v.UintpSlice = utils.SliceMapping(v.UintSlice, utils.AnyPtr[uint]) })
	setter("Uint8pSlice", func() { v.Uint8pSlice = utils.SliceMapping(v.Uint8Slice, utils.AnyPtr[uint8]) })
	setter("Uint16pSlice", func() { v.Uint16pSlice = utils.SliceMapping(v.Uint16Slice, utils.AnyPtr[uint16]) })
	setter("Uint32pSlice", func() { v.Uint32pSlice = utils.SliceMapping(v.Uint32Slice, utils.AnyPtr[uint32]) })
	setter("Uint64pSlice", func() { v.Uint64pSlice = utils.SliceMapping(v.Uint64Slice, utils.AnyPtr[uint64]) })

	setter("Float32p", func() { v.Float32p = utils.AnyPtr(v.Float32) })
	setter("Float64p", func() { v.Float64p = utils.AnyPtr(v.Float64) })
	setter("Float32pSlice", func() { v.Float32pSlice = utils.SliceMapping(v.Float32Slice, utils.AnyPtr[float32]) })
	setter("Float64pSlice", func() { v.Float64pSlice = utils.SliceMapping(v.Float64Slice, utils.AnyPtr[float64]) })

	setter("Complex64", func() { v.Complex64 = complex(1, 1) })
	setter("Complex64p", func() { v.Complex64p = utils.AnyPtr(v.Complex64) })
	setter("Complex128", func() { v.Complex128 = complex(1, 1) })
	setter("Complex128p", func() { v.Complex128p = utils.AnyPtr(v.Complex128) })
	setter("Complex64Slice", func() {
		v.Complex64Slice = []complex64{complex(1, 1), complex(-1, 1), complex(1, -1), complex(-1, -1)}
	})
	setter("Complex64Slice", func() {
		v.Complex64pSlice = utils.SliceMapping(v.Complex64Slice, utils.AnyPtr[complex64])
	})

	setter("Complex128Slice", func() {
		v.Complex128Slice = []complex128{complex(1, 1), complex(-1, 1), complex(1, -1), complex(-1, -1)}
	})
	setter("Complex128pSlice", func() {
		v.Complex128pSlice = utils.SliceMapping(v.Complex128Slice, utils.AnyPtr[complex128])
	})
	setter("Uintptr", func() { v.Uintptr = uintptr(unsafe.Pointer(&v)) })
	setter("Uintptrp", func() { v.Uintptrp = utils.AnyPtr(v.Uintptr) })
	setter("UintptrSlice", func() {
		v.UintptrSlice = []uintptr{uintptr(unsafe.Pointer(&v)),
			uintptr(unsafe.Pointer(&v.Int)), uintptr(unsafe.Pointer(&v.Uint))}
	})
	setter("UintptrpSlice", func() { v.UintptrpSlice = utils.SliceMapping(v.UintptrSlice, utils.AnyPtr[uintptr]) })

	setter("Any", func() { v.Any = "any" })
	setter("Anyp", func() { v.Anyp = utils.AnyPtr(v.Any) })
	setter("AnySlice", func() { v.AnySlice = []any{1, 1.1, "any", true, complex(1, 1), nil} })
	setter("AnypSlice", func() { v.AnypSlice = utils.SliceMapping(v.AnySlice, utils.AnyPtr[any]) })

	setter("Timep", func() { v.Timep = utils.AnyPtr(v.Time) })

	return
}

func GenObj[T any](opts ...utils.OptionExtender) (v T) {
	opt := utils.ApplyOptions[randomOption](opts...)
	utils.MustSuccess(faker.FakeData(&v,
		options.WithNilIfLenIsZero(true),
		options.WithGenerateUniqueValues(true),
		options.WithRandomIntegerBoundaries(interfaces.RandomIntegerBoundary{Start: 1, End: math.MaxInt8}),
		options.WithRandomFloatBoundaries(interfaces.RandomFloatBoundary{Start: 1, End: math.MaxInt8}),
		options.WithFieldsToIgnore(opt.ignored...),
	))
	return
}

func GenObjList[T any](num int, opts ...utils.OptionExtender) (vList []T) {
	vList = make([]T, 0, num)
	for i := 0; i < num; i++ {
		vList = append(vList, GenObj[T](opts...))
	}
	return
}
