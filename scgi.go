package gofcgisrv

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
)

type SCGIRequester struct {
	applicationAddr string
}

func (sr *SCGIRequester) Request(env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	// Make a connection
	conn, err := net.Dial("tcp", sr.applicationAddr)
	if err != nil {
		return err
	}

	// Send the environment
	header := bytes.NewBuffer(nil)
	// Content length has to go first.
	for _, envstring := range env {
		splits := strings.SplitN(envstring, "=", 2)
		if len(splits) == 2 && splits[0] == "CONTENT_LENGTH" {
			fmt.Fprintf(header, "%s\000%s\000", splits[0], splits[1])
		}
	}
	io.WriteString(header, "SCGI:\x001\x00")
	for _, envstring := range env {
		splits := strings.SplitN(envstring, "=", 2)
		if len(splits) == 2 && splits[0] != "CONTENT_LENGTH" {
			fmt.Fprintf(header, "%s\000%s\000", splits[0], splits[1])
		}
	}

	_, err = fmt.Fprintf(conn, "%d:%s,", header.Len(), header.Bytes())
	if err != nil {
		conn.Close()
		return err
	}
	_, err = io.Copy(conn, stdin)
	if err != nil {
		conn.Close()
		return err
	}

	// Flup needs the write side closed. I don't think that's right, but there it is.
	if cw, ok := conn.(interface {
		CloseWrite() error
	}); ok {
		cw.CloseWrite()
	}
	_, err = io.Copy(stdout, conn)
	// If we have an error, just log it to stderr.
	if err != nil {
		stderr.Write([]byte(err.Error()))
	}
	return nil
}

func NewSCGI(addr string) *SCGIRequester {
	return &SCGIRequester{applicationAddr: addr}
}
