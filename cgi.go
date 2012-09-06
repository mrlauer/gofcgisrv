package gofcgisrv

import (
	"io"
	"os/exec"
)

// A CGI server.
type CGIRequester struct {
	cmd  string
	args []string
}

func (cr *CGIRequester) Request(env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := exec.Command(cr.cmd, cr.args...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return err
}

func NewCGI(cmd string, args ...string) *CGIRequester {
	return &CGIRequester{cmd: cmd, args: args}
}
