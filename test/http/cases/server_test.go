package cases

import (
	"context"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"github.com/wfusion/gofusion/log"

	testHtp "github.com/wfusion/gofusion/test/http"
)

func TestServer(t *testing.T) {
	testingSuite := &Server{Test: new(testHtp.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Server struct {
	*testHtp.Test
}

func (t *Server) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Server) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Server) TestStartAndStop() {
	t.Catch(func() {
		engine := t.ServerGiven(http.MethodGet, "/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, map[string]any{"code": 0, "msg": "ok"})
		})

		wg := new(sync.WaitGroup)
		wg.Add(1)
		go func() {
			defer wg.Done()
			t.Require().NoError(engine.ListenAndServe())
		}()

		time.Sleep(time.Second)

		process, err := os.FindProcess(os.Getpid())
		t.Require().NoError(err)
		t.Require().NoError(process.Signal(os.Interrupt))

		wg.Wait()
	})
}
