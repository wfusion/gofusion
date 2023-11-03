package gracefully

import (
	"net/http"

	_ "unsafe"
)

//go:linkname isClosedConnError golang.org/x/net/http2.isClosedConnError
// isClosedConnError reports whether err is an error from use of a closed
// network connection.
func isClosedConnError(err error) bool

//go:linkname setupHTTP2_Serve net/http.(*Server).setupHTTP2_Serve
func setupHTTP2_Serve(srv *http.Server) error

//go:linkname setupHTTP2_ServeTLS net/http.(*Server).setupHTTP2_ServeTLS
func setupHTTP2_ServeTLS(srv *http.Server) error
