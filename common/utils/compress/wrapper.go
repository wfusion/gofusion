package compress

import (
	"io"
	
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/zlib"
)

type s2Decodable struct {
	*s2.Reader
}

func (s *s2Decodable) Read(p []byte) (n int, err error) { return s.Reader.Read(p) }
func (s *s2Decodable) Reset(r io.Reader) (err error)    { s.Reader.Reset(r); return }

type gzipDecodable struct {
	io.ReadCloser
}

func (g *gzipDecodable) Read(p []byte) (n int, err error) { return g.ReadCloser.Read(p) }
func (g *gzipDecodable) Reset(r io.Reader) (err error)    { g.ReadCloser, err = gzip.NewReader(r); return }

type deflateDecodable struct {
	io.ReadCloser
}

func (d *deflateDecodable) Read(p []byte) (n int, err error) { return d.ReadCloser.Read(p) }
func (d *deflateDecodable) Reset(r io.Reader) (err error)    { d.ReadCloser = flate.NewReader(r); return }

type zlibDecodable struct {
	io.ReadCloser
}

func (z *zlibDecodable) Read(p []byte) (n int, err error) { return z.ReadCloser.Read(p) }
func (z *zlibDecodable) Reset(r io.Reader) (err error)    { z.ReadCloser, err = zlib.NewReader(r); return }
