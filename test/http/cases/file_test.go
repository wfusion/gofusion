package cases

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/env"
	"github.com/wfusion/gofusion/log"

	fmkHtp "github.com/wfusion/gofusion/http"
	testHtp "github.com/wfusion/gofusion/test/http"
)

func TestFile(t *testing.T) {
	testingSuite := &File{Test: new(testHtp.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type File struct {
	*testHtp.Test
}

func (t *File) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *File) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *File) TestStatic() {
	t.Catch(func() {
		// Given
		files := t.ConfigFiles()
		p := filepath.Join(env.WorkDir, fmt.Sprintf("/configs/%s.%s", t.AppName(), files[len(files)-1]))
		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/TestStatic.yml", nil)
		t.NoError(err)
		engine := t.ServerGiven("File", "/TestStatic.yml", p)

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.NotEmpty(w.Body.String())
	})
}

func (t *File) TestStaticZeroCopy() {
	t.Catch(func() {
		// Given
		files := t.ConfigFiles()
		p := filepath.Join(env.WorkDir, fmt.Sprintf("/configs/%s.%s", t.AppName(), files[len(files)-1]))
		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/TestStaticZeroCopy.yml", nil)
		t.NoError(err)
		engine := t.ServerGiven("File", "/TestStaticZeroCopyMock.yml", p)
		engine.GET("/TestStaticZeroCopy.yml", fmkHtp.StaticFileZeroCopy(p))

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.NotEmpty(w.Body.String())
	})
}

func (t *File) TestContentZeroCopy() {
	t.Catch(func() {
		// Given
		files := t.ConfigFiles()
		p := filepath.Join(env.WorkDir, fmt.Sprintf("/configs/%s.%s", t.AppName(), files[len(files)-1]))
		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/TestContentZeroCopy.yml", nil)
		t.NoError(err)
		engine := t.ServerGiven("File", "/TestContentZeroCopyMock.yml", p)
		engine.GET("/TestContentZeroCopy.yml", fmkHtp.ContentZeroCopy(func(c *gin.Context) (
			name string, modTime time.Time, content io.ReadSeeker, err error) {
			f, err := os.Open(p)
			if err != nil {
				return
			}
			s, err := f.Stat()
			if err != nil {
				return
			}
			return f.Name(), s.ModTime(), f, nil
		}))

		// When
		engine.ServeHTTP(w, req)

		// Then
		t.Equal(http.StatusOK, w.Code)
		t.NotEmpty(w.Body.String())
	})
}
