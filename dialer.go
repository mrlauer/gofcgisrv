package gofcgisrv

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
)

type Dialer interface {
	Dial() (net.Conn, error)
}

type NetDialer struct {
	net  string
	addr string
}

func (d NetDialer) Dial() (net.Conn, error) {
	return net.Dial(d.net, d.addr)
}

// StdinDialer managers an app as a child process, creating a socket and passing it through stdin.
type StdinDialer struct {
	app      string
	args     []string
	cmd      *exec.Cmd
	stdin    *os.File
	listener net.Listener
	filename string
}

func (sd *StdinDialer) Dial() (net.Conn, error) {
	if sd.stdin == nil {
		return nil, errors.New("No file")
	}
	return net.Dial("unix", sd.filename)
}

func (sd *StdinDialer) Start() error {
	// Create a socket.
	// We'll use the high-level net API, creating a listener that does all sorts
	// of socket stuff, getting its file, and passing that (really just for its FD)
	// to the child process.
	// We'll rely on crypt/rand to get a unique filename for the socket.
	tmpdir := os.TempDir()
	rnd := make([]byte, 8)
	n, err := rand.Read(rnd)
	if err != nil {
		return err
	}
	basename := fmt.Sprintf("fcgi%x", rnd[:n])
	filename := path.Join(tmpdir, basename)

	listener, err := net.Listen("unix", filename)
	if err != nil {
		return err
	}
	socket, err := listener.(*net.UnixListener).File()
	if err != nil {
		listener.Close()
		return err
	}
	cmd := exec.Command(sd.app, sd.args...)
	cmd.Stdin = socket
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		socket.Close()
		listener.Close()
		return err
	}
	sd.stdin = socket
	sd.listener = listener
	sd.cmd = cmd
	sd.filename = filename
	return nil
}

func (sd *StdinDialer) Close() {
	if sd.stdin != nil {
		sd.stdin.Close()
		sd.stdin = nil
	}
	if sd.listener != nil {
		sd.listener.Close()
		sd.listener = nil
	}
	if sd.cmd != nil {
		sd.cmd.Process.Kill()
		sd.cmd = nil
	}
	sd.filename = ""
}
