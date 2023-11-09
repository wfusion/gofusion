package http

import (
	"io"
	"net"
	"net/http"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/multierr"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/http/gracefully"
)

type getContentFn func(c *gin.Context) (name string, modTime time.Time, content io.ReadSeeker, err error)

// StaticFileZeroCopy zero copy gin handler wrapper for static file
func StaticFileZeroCopy(filename string) func(c *gin.Context) {
	filename = path.Clean(filename)
	return func(c *gin.Context) {
		http.ServeFile(&ginZeroCopyWriter{ResponseWriter: c.Writer, ctx: c}, c.Request, filename)
	}
}

// ContentZeroCopy zero copy gin handler wrapper for seeker
func ContentZeroCopy(fn getContentFn, opts ...utils.OptionExtender) func(c *gin.Context) {
	opt := utils.ApplyOptions[useOption](opts...)
	return func(c *gin.Context) {
		name, modTime, content, err := fn(c)
		if err != nil {
			code, data, page, count, msg := parseRspError(nil, err)
			rspError(c, opt.appName, code, data, page, count, msg)
			c.Abort()
			return
		}
		defer utils.CloseAnyway(content)
		http.ServeContent(&ginZeroCopyWriter{ResponseWriter: c.Writer, ctx: c}, c.Request, name, modTime, content)
	}
}

type ginZeroCopyWriter struct {
	gin.ResponseWriter

	ctx *gin.Context
}

func (z *ginZeroCopyWriter) ReadFrom(r io.Reader) (n int64, err error) {
	var size int64
	if limitedReader, ok := r.(*io.LimitedReader); ok {
		size = limitedReader.N
	}

	// forces to write the http header (status code + headers)
	z.ResponseWriter.WriteHeaderNow()
	if z.ctx.Request.Method == http.MethodHead {
		return size, nil
	}

	// hijack conn to call zero copy
	var conn net.Conn
	_, err = utils.Catch(func() (err error) { conn, _, err = z.ResponseWriter.Hijack(); return })
	if err != nil || conn == nil {
		// write by memory buffer
		if size > 0 {
			return io.CopyN(z.ResponseWriter, r.(*io.LimitedReader), size)
		}
		return io.Copy(z.ResponseWriter, r)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			err = multierr.Append(err, closeErr)
		}
	}()

	// set write timeout again because it is reset when hijack
	if err = conn.SetWriteDeadline(time.Now().Add(gracefully.DefaultWriteTimeOut)); err != nil {
		return
	}

	// write body
	return io.Copy(conn, r)
}
