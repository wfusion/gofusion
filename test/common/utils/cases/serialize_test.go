package cases

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wfusion/gofusion/test/internal/mock"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/log"
	testUtl "github.com/wfusion/gofusion/test/common/utils"
)

func TestSerialize(t *testing.T) {
	t.Parallel()
	testingSuite := &Serialize{Test: new(testUtl.Test)}
	suite.Run(t, testingSuite)
}

type Serialize struct {
	*testUtl.Test
}

func (t *Serialize) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Serialize) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Serialize) TestBytes() {
	t.Catch(func() {
		algos := []serialize.Algorithm{
			serialize.AlgorithmGob,
			serialize.AlgorithmJson,
			serialize.AlgorithmMsgpack,
			serialize.AlgorithmCbor,
		}

		for _, algo := range algos {
			var (
				unmarshalSingleFunc   func([]byte) (any, error)
				unmarshalMultipleFunc func([]byte) (any, error)
			)

			expected := mock.GenObjBySerializeAlgo(algo)
			expectedList := mock.GenObjListBySerializeAlgo(algo, 3)
			switch expected.(type) {
			case *mock.CommonObj:
				unmarshalSingleFunc = func(s []byte) (any, error) {
					return serialize.UnmarshalFunc[*mock.CommonObj](algo)(s)
				}
				unmarshalMultipleFunc = func(s []byte) (any, error) {
					return serialize.UnmarshalFunc[[]*mock.CommonObj](algo)(s)
				}
			case *mock.RandomObj:
				unmarshalSingleFunc = func(s []byte) (any, error) {
					return serialize.UnmarshalFunc[*mock.RandomObj](algo)(s)
				}
				unmarshalMultipleFunc = func(s []byte) (any, error) {
					return serialize.UnmarshalFunc[[]*mock.RandomObj](algo)(s)
				}
			}
			marshalFunc := serialize.MarshalFunc(algo)
			t.Run(algo.String(), func() {
				// single
				marshaled, err := marshalFunc(expected)
				t.Require().NoError(err)
				actualSingle, err := unmarshalSingleFunc(marshaled)
				t.Require().NoError(err)
				t.Require().EqualValues(expected, actualSingle)

				// multiple
				marshaled, err = marshalFunc(expectedList)
				t.Require().NoError(err)
				actualMultiple, err := unmarshalMultipleFunc(marshaled)
				t.Require().NoError(err)
				t.Require().EqualValues(expectedList, actualMultiple)
			})
		}
	})
}

func (t *Serialize) TestStream() {
	t.Catch(func() {
		algos := []serialize.Algorithm{
			serialize.AlgorithmGob,
			serialize.AlgorithmJson,
			serialize.AlgorithmMsgpack,
			serialize.AlgorithmCbor,
		}
		commonType := reflect.TypeOf((*mock.CommonObj)(nil))
		commonListType := reflect.SliceOf(commonType)
		randomType := reflect.TypeOf((*mock.RandomObj)(nil))
		randomListType := reflect.SliceOf(randomType)

		for _, algo := range algos {
			var (
				unmarshalSingleFunc   func(io.Reader) (any, error)
				unmarshalMultipleFunc func(io.Reader) (any, error)
			)
			expected := mock.GenObjBySerializeAlgo(algo)
			expectedList := mock.GenObjListBySerializeAlgo(algo, 3)
			switch expected.(type) {
			case *mock.CommonObj:
				unmarshalSingleFunc = serialize.UnmarshalStreamFuncByType(algo, commonType)
				unmarshalMultipleFunc = serialize.UnmarshalStreamFuncByType(algo, commonListType)
			case *mock.RandomObj:
				unmarshalSingleFunc = serialize.UnmarshalStreamFuncByType(algo, randomType)
				unmarshalMultipleFunc = serialize.UnmarshalStreamFuncByType(algo, randomListType)
			}

			marshalFunc := serialize.MarshalStreamFunc(algo)
			t.Run(algo.String(), func() {
				marshaled, cb := utils.BytesBufferPool.Get(nil)
				defer cb()

				// single
				err := marshalFunc(marshaled, expected)
				t.Require().NoError(err)
				actualSingle, err := unmarshalSingleFunc(marshaled)
				t.Require().NoError(err)
				t.Require().EqualValues(expected, actualSingle)

				marshaled.Reset()

				// multiple
				err = marshalFunc(marshaled, expectedList)
				t.Require().NoError(err)
				actualMultiple, err := unmarshalMultipleFunc(marshaled)
				t.Require().NoError(err)
				t.Require().EqualValues(expectedList, actualMultiple)
			})
		}
	})
}
