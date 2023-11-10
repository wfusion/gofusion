package middleware

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"golang.org/x/text/language"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/i18n"

	fmkCtx "github.com/wfusion/gofusion/context"
)

func Recover(appName string, logger resty.Logger) gin.HandlerFunc {
	tag := i18n.DefaultLang(i18n.AppName(appName))
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}
				if brokenPipe {
					c.Abort()
					return
				}
				debugStack := ""
				for _, v := range strings.Split(string(debug.Stack()), "\n") {
					debugStack += "> " + v + "\n"
				}
				hostname, hostnameErr := os.Hostname()
				if hostnameErr != nil {
					hostname = "unknown"
				}
				buffer, cb := utils.BytesBufferPool.Get(nil)
				defer cb()

				if tag == language.Chinese {
					buffer.WriteString(fmt.Sprintf("%v \n", err))
					buffer.WriteString(fmt.Sprintf("请求时间：%v \n", time.Now().Format(constant.StdTimeLayout)))
					buffer.WriteString(fmt.Sprintf("主机名称：%v \n", hostname))
					buffer.WriteString(fmt.Sprintf("请求编号：%v \n", c.GetString(fmkCtx.KeyTraceID)))
					buffer.WriteString(fmt.Sprintf("请求地址：%v \n",
						c.Request.Method+"  "+c.Request.Host+c.Request.RequestURI))
					buffer.WriteString(fmt.Sprintf("请求UA：%v \n", c.Request.UserAgent()))
					buffer.WriteString(fmt.Sprintf("请求IP：%v \n", c.ClientIP()))
					buffer.WriteString(fmt.Sprintf("异常捕获：\n%v", debugStack))
				} else {
					buffer.WriteString(fmt.Sprintf("%v \n", err))
					buffer.WriteString(fmt.Sprintf("RequstTime：%v \n", time.Now().Format(constant.StdTimeLayout)))
					buffer.WriteString(fmt.Sprintf("Hostname：%v \n", hostname))
					buffer.WriteString(fmt.Sprintf("TraceID：%v \n", c.GetString(fmkCtx.KeyTraceID)))
					buffer.WriteString(fmt.Sprintf("RequestURI：%v \n",
						c.Request.Method+"  "+c.Request.Host+c.Request.RequestURI))
					buffer.WriteString(fmt.Sprintf("UA：%v \n", c.Request.UserAgent()))
					buffer.WriteString(fmt.Sprintf("IP：%v \n", c.ClientIP()))
					buffer.WriteString(fmt.Sprintf("ErrorStack：\n%v", debugStack))
				}

				if logger != nil {
					logger.Errorf(buffer.String(), fmkCtx.New(fmkCtx.Gin(c)))
				} else {
					log.Printf(buffer.String())
				}

				c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]any{
					"code":    http.StatusInternalServerError,
					"message": "service internal error",
				})
			}
		}()
		c.Next()
	}
}
