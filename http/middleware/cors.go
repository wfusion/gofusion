package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		var headerKeys []string
		for k := range c.Request.Header {
			headerKeys = append(headerKeys, k)
		}
		corsHeader := c.GetHeader("Access-Control-Request-Headers")
		headerKeys = append(headerKeys, strings.Split(corsHeader, ",")...)

		headerStr := strings.Join(headerKeys, ", ")
		if headerStr != "" {
			headerStr = fmt.Sprintf("access-control-allow-origin, access-control-allow-headers, %s", headerStr)
		} else {
			headerStr = "access-control-allow-origin, access-control-allow-headers"
		}
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Headers", headerStr)
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
			c.Writer.Header().Set("Access-Control-Expose-Headers",
				"Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatusJSON(http.StatusNoContent, "no content")
		}
		c.Next()
	}
}
