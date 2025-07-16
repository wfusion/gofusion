package cases

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"

	testLog "github.com/wfusion/gofusion/test/log"
)

func TestRotate(t *testing.T) {
	testingSuite := &Rotate{Test: new(testLog.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Rotate struct {
	*testLog.Test
}

func (t *Rotate) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Rotate) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Rotate) TestRotateSize() {
	t.Catch(func() {
		// Given
		msg := utils.RandomLetterAndNumber(1024 - 72)
		logger := log.Use("rotate_size", log.AppName(t.AppName()))

		// When
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		for i := 0; i < 10; i++ {
			logger.Info(ctx, "%v:"+msg, i)
		}

		// Then
		_, filename, _, ok := runtime.Caller(0)
		t.Require().True(ok)
		projectRoot := filepath.Dir(filename)

		time.Sleep(time.Second)
		matches, err := filepath.Glob(filepath.Join(projectRoot, "gofusion*.log"))
		t.Require().NoError(err)
		t.Greater(len(matches), 1)
		t.LessOrEqual(len(matches), 1+5)
		for _, match := range matches {
			fs, err := os.Stat(match)
			t.Require().NoError(err)
			t.LessOrEqual(fs.Size(), int64(1024))
			t.Require().NoError(os.Remove(match))
		}
	})
}

func (t *Rotate) TestRotateTime() {
	t.Catch(func() {
		// Given
		msg := utils.RandomLetterAndNumber(1024 - 72)
		logger := log.Use("rotate_time", log.AppName(t.AppName()))

		// When
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		for i := 0; i < 3; i++ {
			logger.Info(ctx, "%v:"+msg, i)
			time.Sleep(time.Second)
		}

		// Then
		_, filename, _, ok := runtime.Caller(0)
		t.Require().True(ok)
		projectRoot := filepath.Dir(filename)

		time.Sleep(time.Second)
		matches, err := filepath.Glob(filepath.Join(projectRoot, "gofusion*.log"))
		t.Require().NoError(err)
		t.Require().Equal(len(matches), 1+1)
		for _, match := range matches {
			fs, err := os.Stat(match)
			t.Require().NoError(err)
			t.NotZero(fs.Size())
			t.Require().NoError(os.Remove(match))
		}
	})
}

func (t *Rotate) TestRotateCount() {
	t.Catch(func() {
		// Given
		msg := utils.RandomLetterAndNumber(1024 - 74)
		logger := log.Use("rotate_count", log.AppName(t.AppName()))

		// When
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		for i := 0; i < 100; i++ {
			logger.Info(ctx, "%v:"+msg, i)
		}

		// Then
		_, filename, _, ok := runtime.Caller(0)
		t.Require().True(ok)
		projectRoot := filepath.Dir(filename)

		time.Sleep(time.Second)
		matches, err := filepath.Glob(filepath.Join(projectRoot, "gofusion*.log"))
		t.Require().NoError(err)
		t.LessOrEqual(len(matches), 1+1)
		for _, match := range matches {
			fs, err := os.Stat(match)
			t.Require().NoError(err)
			t.LessOrEqual(fs.Size(), int64(1024))
			t.Require().NoError(os.Remove(match))
		}
	})
}
