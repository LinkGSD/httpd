package httpd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type chunkReader struct {
	n    int
	bufr *bufio.Reader
	done bool
	crlf [2]byte
}

func (cw *chunkReader) Read(p []byte) (n int, err error) {
	if cw.done {
		return 0, io.EOF
	}
	if cw.n == 0 {
		cw.n, err = cw.getChunkSize()
		if err != nil {
			return
		}
	}
	if cw.n == 0 {
		cw.done = true
		err = cw.discardCRLF()
		return
	}
	if len(p) <= cw.n {
		n, err = cw.bufr.Read(p)
		cw.n -= n
		return
	}

	n, _ = io.ReadFull(cw.bufr, p[:cw.n])
	cw.n = 0
	if err = cw.discardCRLF(); err != nil {
		return
	}
	return
}

func (cw *chunkReader) discardCRLF() (err error) {
	if _, err = io.ReadFull(cw.bufr, cw.crlf[:]); err != nil {
		if cw.crlf[0] != '\r' || cw.crlf[1] != '\n' {
			return errors.New("unsupported encoding format of chunk")
		}
	}
	return
}

func (cw *chunkReader) getChunkSize() (chunkSize int, err error) {
	line, err := readLine(cw.bufr)
	if err != nil {
		return
	}

	for i := 0; i < len(line); i++ {
		switch {
		case 'a' <= line[i] && line[i] <= 'f':
			chunkSize = chunkSize*16 + int(line[i]-'a') + 10
		case 'A' <= line[i] && line[i] <= 'F':
			chunkSize = chunkSize*16 + int(line[i]-'A') + 10
		case '0' <= line[i] && line[i] <= '9':
			chunkSize = chunkSize*16 + int(line[i]-'0')
		default:
			return 0, errors.New("illegal hex number")
		}
	}
	return
}

type chunkWriter struct {
	resp  *response
	wrote bool
}

func (cw *chunkWriter) Write(p []byte) (n int, err error) {
	if !cw.wrote {
		cw.finalizeHeader(p)
		if err = cw.writeHeader(); err != nil {
			return
		}
		cw.wrote = true
	}
	bufw := cw.resp.bufw
	if cw.resp.chunking {
		_, err = fmt.Fprintf(bufw, "%x\r\n", len(p))
		if err != nil {
			return
		}
	}
	n, err = bufw.Write(p)
	if err == nil && cw.resp.chunking {
		_, err = bufw.WriteString("\r\n")
	}
	return
}

func (cw *chunkWriter) finalizeHeader(p []byte) {
	header := cw.resp.header
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", http.DetectContentType(p))
	}

	if header.Get("Content-Length") == "" && header.Get("Transfer-Encoding") == "" {
		if cw.resp.handlerDone {
			buffered := cw.resp.bufw.Buffered()
			header.Set("Content-Length", strconv.Itoa(buffered))
		} else {
			cw.resp.chunking = true
			header.Set("Transfer-Encoding", "chunked")
		}
		return
	}

	if header.Get("Transfer-Encoding") == "chunked" {
		cw.resp.chunking = true
	}
}

func (cw *chunkWriter) writeHeader() error {
	codeString := strconv.Itoa(cw.resp.statusCode)
	statusLine := cw.resp.req.Proto + " " + codeString + " " + statusText[cw.resp.statusCode] + "\r\n"
	bufw := cw.resp.c.bufw
	_, err := bufw.WriteString(statusLine)
	if err != nil {
		return err
	}
	for k, v := range cw.resp.header {
		_, err = bufw.WriteString(k + ": " + v[0] + "\r\n")
		if err != nil {
			return err
		}
	}
	_, err = bufw.WriteString("\r\n")
	return err
}
