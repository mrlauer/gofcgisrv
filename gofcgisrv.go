/*
Package gofcgisrv implements the webserver side of the FastCGI protocol.
*/
package gofcgisrv

import (
	"net"
	"sync"
	"io"
)


// Server is the external interface. It manages connections to a single FastCGI application.
// A server may maintain many connections, each of which may multiplex many requests.
type Server struct {
	applicationAddr string
	connections     []*conn
	connLock        sync.RWMutex

	// Parameters of the application
	mpx         bool
	maxConns    int
	maxRequests int
}

func NewServer(applicationAddr string) *Server {
	s := &Server{applicationAddr: applicationAddr}
	s.maxConns = 1
	s.maxRequests = 1
	return s
}

// Request executes a request using env and stdin as inputs and stdout and stderr as outputs.
// env should be a slice of name=value pairs. It blocks until the application has finished.
func (s *Server) Request(env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	return nil
}

// Conn wraps a net.Conn. It may multiplex many requests.
type conn struct {
	netconn  net.Conn
	requests []*request
	reqLock  sync.RWMutex
}

// Request is a single request.
type request struct {
	id   requestId
	conn *conn
}

