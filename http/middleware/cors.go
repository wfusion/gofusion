package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
)

func Cors(origins, methods, headers, exposeHeaders []string, credential, options, forbidden string) gin.HandlerFunc {
	allowOrigins := utils.NewSet(origins...)

	if len(methods) == 0 {
		methods = []string{"POST", "OPTIONS", "GET", "PUT", "DELETE"}
	}
	allowMethods := strings.Join(methods, ", ")

	allowHeaders := ""
	if len(headers) > 0 {
		allowHeaders = fmt.Sprintf("access-control-allow-origin, access-control-allow-headers, %s",
			strings.Join(headers, ", "))
	}

	exposeHeader := "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type"
	if len(exposeHeaders) > 0 {
		exposeHeader = strings.Join(exposeHeaders, ", ")
	}

	allowCredentials := true
	if credential != "" {
		allowCredentials = cast.ToBool(credential)
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" && allowOrigins.Size() > 0 && !allowOrigins.Contains(origin) {
			if forbidden == "" {
				c.AbortWithStatus(http.StatusForbidden)
			} else {
				c.AbortWithStatusJSON(http.StatusForbidden, forbidden)
			}
			return
		}

		if allowHeaders == "" {
			corsHeader := c.GetHeader("Access-Control-Request-Headers")
			requestHeaderKeys := make([]string, 0, len(c.Request.Header))
			for k := range c.Request.Header {
				requestHeaderKeys = append(requestHeaderKeys, k)
			}
			requestHeaderKeys = append(requestHeaderKeys, strings.Split(corsHeader, ",")...)
			if len(requestHeaderKeys) > 0 {
				allowHeaders = fmt.Sprintf("access-control-allow-origin, access-control-allow-headers, %s",
					strings.Join(requestHeaderKeys, ", "))
			} else {
				allowHeaders = "access-control-allow-origin, access-control-allow-headers"
			}
		}

		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Methods", allowMethods)
			c.Writer.Header().Set("Access-Control-Expose-Headers", exposeHeader)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", cast.ToString(allowCredentials))
			if allowHeaders != "" {
				c.Writer.Header().Set("Access-Control-Allow-Headers", allowHeaders)
			}
		}

		if c.Request.Method == "OPTIONS" {
			if options == "" {
				c.AbortWithStatus(http.StatusNoContent)
			} else {
				c.AbortWithStatusJSON(http.StatusNoContent, options)
			}
			return
		}
		c.Next()
	}
}
