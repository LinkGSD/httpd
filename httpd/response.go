package httpd

import (
	"bufio"
	"fmt"
)

type response struct {
	c *conn

	wroteHeader bool
	header      Header

	statusCode int

	handlerDone bool

	bufw *bufio.Writer
	cw   *chunkWriter

	req *Request

	closeAfterReply bool

	chunking bool
}

type ResponseWriter interface {
	Write([]byte) (n int, err error)
	Header() Header
	WriteHeader(statusCode int)
}

func setupResponse(c *conn, req *Request) *response {
	resp := &response{
		c:          c,
		header:     make(Header),
		statusCode: 200,
		req:        req,
	}
	cw := &chunkWriter{resp: resp}
	resp.cw = cw
	resp.bufw = bufio.NewWriterSize(cw, bufSize)
	var (
		protoMinor int
		protoMajor int
	)
	fmt.Sscanf(req.Proto, "HTTP/%d%d", &protoMinor, &protoMajor)
	if protoMinor < 1 || protoMinor == 1 && protoMajor == 0 || req.Header.Get("Connectioni") == "close" {
		resp.closeAfterReply = true
	}
	return resp
}

func (w *response) Write(p []byte) (int, error) {
	n, err := w.c.bufw.Write(p)
	if err != nil {
		w.closeAfterReply = true
	}
	return n, err
}

func (w *response) Header() Header {
	return w.header
}

func (w *response) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.statusCode = statusCode
	w.wroteHeader = true
}
