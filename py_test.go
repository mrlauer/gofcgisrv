// +build all

package gofcgisrv

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestPyServer(t *testing.T) {
	cmd := exec.Command("python", "./testdata/cgi_test.py", "--port=9001")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Error running cgi_test.py: %v", err)
	}
	defer time.Sleep(time.Millisecond * 10)
	defer cmd.Process.Kill()

	time.Sleep(time.Millisecond * 100)
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
	time.Sleep(time.Millisecond * 90)
	s := NewCGI("python", "./testdata/cgi_test.py", "--cgi")
	testRequester(t, httpTestData{
		name:     "py cgi",
		f:        s,
		body:     strings.NewReader("This is a test"),
		status:   200,
		expected: "This is a test",
	})
}
