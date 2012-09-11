// +build !smoke

package gofcgisrv

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func waitForConn(addr string, timeout time.Duration) error {
	ticker := time.NewTicker(time.Millisecond * 10)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c, err := net.Dial("tcp", addr)
			if err == nil {
				c.Close()
				return nil
			}
		case <-timer.C:
			return errors.New("timeout")
		}
	}
	panic("Unreachable")
}

func TestPyServer(t *testing.T) {
	cmd := exec.Command("python", "./testdata/cgi_test.py", "--port=9001")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Error running cgi_test.py: %v", err)
	}
	defer cmd.Process.Kill()

	waitForConn("127.0.0.1:9001", time.Second)
	s := NewServer("127.0.0.1:9001")
	testRequester(t, httpTestData{
		name:     "py fastcgi",
		f:        s,
		body:     strings.NewReader("This is a test"),
		status:   200,
		expected: "This is a test",
	})
}

func TestPyCGI(t *testing.T) {
	s := NewCGI("python", "./testdata/cgi_test.py", "--cgi")
	testRequester(t, httpTestData{
		name:     "py cgi",
		f:        s,
		body:     strings.NewReader("This is a test"),
		status:   200,
		expected: "This is a test",
	})
}

func TestPySCGI(t *testing.T) {
	cmd := exec.Command("python", "./testdata/cgi_test.py", "--scgi", "--port=9002")
	// flup barfs some output. Why?? Seems wrong to me.
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Error running cgi_test.py: %v", err)
	}
	defer cmd.Process.Kill()
	waitForConn("127.0.0.1:9002", time.Second)
	s := NewSCGI("127.0.0.1:9002")
	testRequester(t, httpTestData{
		name:     "py scgi",
		f:        s,
		body:     strings.NewReader("This is a test"),
		status:   200,
		expected: "This is a test",
	})
}
