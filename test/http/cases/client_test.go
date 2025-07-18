package cases

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/log"

	fusHtp "github.com/wfusion/gofusion/http"
	testHtp "github.com/wfusion/gofusion/test/http"
)

func TestClient(t *testing.T) {
	testingSuite := &Client{Test: new(testHtp.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Client struct {
	*testHtp.Test
}

func (t *Client) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)

		httpmock.Activate()
	})
}

func (t *Client) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)

		httpmock.DeactivateAndReset()
	})
}

func (t *Client) TestMock() {
	t.Catch(func() {
		// Given
		fakeUrl := "http://localhost/TestMock"
		expected := &fusHtp.Response{
			Code:    0,
			Message: "ok",
			Data:    2,
		}
		actual := new(fusHtp.Response)
		responder, err := httpmock.NewJsonResponder(http.StatusOK, expected)
		t.Require().NoError(err)
		httpmock.RegisterResponder(http.MethodGet, fakeUrl, responder)
		cli := fusHtp.NewRequest(context.Background(), fusHtp.AppName(t.AppName())).SetResult(&actual)

		// When
		rsp, err := cli.Get(fakeUrl)

		// Then
		t.Require().NoError(err)
		t.Require().Equal(http.StatusOK, rsp.StatusCode())
		t.Require().EqualValues(expected.Data, actual.Data)
	})
}
