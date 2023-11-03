package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/http/consts"
	"github.com/wfusion/gofusion/log"

	fmkCtx "github.com/wfusion/gofusion/context"
)

func logging(c *gin.Context, logger log.Logable, rawURL *url.URL, appName string) {
	ctx := fmkCtx.New(fmkCtx.Gin(c))
	cost := float64(consts.GetReqCost(c)) / float64(time.Millisecond)
	status := c.Writer.Status()
	fields := log.Fields{
		"path":        rawURL.Path,
		"method":      c.Request.Method,
		"status":      status,
		"referer":     c.Request.Referer(),
		"req_size":    c.Request.ContentLength,
		"rsp_size":    c.Writer.Size(),
		"cost":        cost,
		"user_agent":  c.Request.UserAgent(),
		"client_addr": c.ClientIP(),
	}

	// skip health check logging
	if rawURL.Path == "/health" && c.Request.Method == http.MethodGet {
		return
	}

	msg := fmt.Sprintf(
		"%s -> %s %s %d %d %d (%.2fms)",
		c.ClientIP(), utils.LocalIP.String(),
		strconv.Quote(fmt.Sprintf("%s %s", c.Request.Method, rawURL)),
		c.Request.ContentLength, c.Writer.Size(), status, cost,
	)

	switch {
	case status < http.StatusBadRequest:
		logger.Info(ctx, msg, fields)
	case status >= http.StatusBadRequest && status < http.StatusInternalServerError:
		logger.Warn(ctx, msg, fields)
	default:
		logger.Error(ctx, msg, fields)
	}

	// TODO: emit metrics
}

func Logging(appName, logInstance string) gin.HandlerFunc {
	logger := log.Use(logInstance, log.AppName(appName))
	return func(c *gin.Context) {
		reqURL := clone.Clone(c.Request.URL)
		defer logging(c, logger, reqURL, appName)

		c.Next()
	}

}
