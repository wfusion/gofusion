package cases

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"

	fusHtp "github.com/wfusion/gofusion/http"
	testHtp "github.com/wfusion/gofusion/test/http"
)

func TestMiddleware(t *testing.T) {
	testingSuite := &Middleware{Test: new(testHtp.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Middleware struct {
	*testHtp.Test
}

func (t *Middleware) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Middleware) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Middleware) TestRecover() {
	t.Catch(func() {
		// Given
		path := "/TestRecover"
		ctx := context.Background()
		router := fusHtp.Use(fusHtp.AppName(t.AppName()))
		router.POST(path, func(c *gin.Context) error {
			panic(errors.New("TestRecover panic"))
		})
		router.Start()
		<-router.Running()
		req := fusHtp.NewRequest(ctx, fusHtp.CName(clientLocalName), fusHtp.AppName(t.AppName()))
		req.SetHeader("Origin", "localhost")
		rsp, err := req.Post(t.addr() + path)
		t.NoError(err)
		t.EqualValues(http.StatusInternalServerError, rsp.StatusCode())
	})
}

func (t *Middleware) addr() string {
	conf := fusHtp.Use(fusHtp.AppName(t.AppName())).Config()
	if conf.TLS {
		return fmt.Sprintf("https://%s:%v", utils.ClientIP(), conf.Port)
	} else {
		return fmt.Sprintf("http://%s:%v", utils.ClientIP(), conf.Port)
	}
}
