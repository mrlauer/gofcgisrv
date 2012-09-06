package gofcgisrv

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func echoRequester(env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {

	body, err := ioutil.ReadAll(stdin)
	if err != nil {
		return err
	}
	bodyLen := len(body)
	// Find the content-length and make sure it matches
	clen := -1
	for _, e := range env {
		n, _ := fmt.Sscanf(e, "CONTENT_LENGTH=%d", &clen)
		if n == 1 {
			break
		}
	}
	if clen == -1 {
		return errors.New("No content-length")
	}
	if clen != bodyLen {
		return fmt.Errorf("CONTENT_LENGTH %d, body length %d", clen, bodyLen)
	}

	io.WriteString(stdout, "Status: 200 OK\r\n")
	fmt.Fprintf(stdout, "Content-Length: %d\r\n", len(body))
	fmt.Fprintf(stdout, "\r\n")
	stdout.Write(body)
	return nil
}

var _brokenReqError string = "There seems to have been some kind of mistake."

func brokenRequester(env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	return errors.New(_brokenReqError)
}

func makeHandler(f RequesterFunc) http.Handler {
	serve := func(w http.ResponseWriter, r *http.Request) {
		env := HTTPEnv(nil, r)
		ServeHTTP(f, env, w, r)
	}
	return http.HandlerFunc(serve)
}

type httpTestData struct {
	name     string
	f        RequesterFunc
	body     io.Reader
	status   int
	expected string
}

func testRequester(t *testing.T, data httpTestData) {
	server := httptest.NewServer(makeHandler(data.f))
	defer server.Close()

	url := server.URL + "/test"
	resp, err := http.Post(url, "text/plain", data.body)
	if err != nil {
		t.Errorf("Error in post %s: %v", data.name, err)
		return
	}
	if resp.StatusCode != data.status {
		t.Errorf("%s: status was %d, not %d", data.name, resp.StatusCode, data.status)
	}
	b, _ := ioutil.ReadAll(resp.Body)
	if string(b) != data.expected {
		t.Errorf("%s: body was %q, not %q", data.name, b, data.expected)
	}
}

func TestHttp(t *testing.T) {
	data := []httpTestData{
		{
			name:     "echo",
			f:        echoRequester,
			body:     strings.NewReader("This is a test"),
			status:   200,
			expected: "This is a test",
		},
		{
			name:     "echo (different reader)",
			f:        echoRequester,
			body:     bufio.NewReader(strings.NewReader("This is a test")),
			status:   200,
			expected: "This is a test",
		},
		{
			name:     "broken",
			f:        brokenRequester,
			body:     strings.NewReader("This is a test"),
			status:   500,
			expected: _brokenReqError + "\n",
		},
	}
	for _, d := range data {
		testRequester(t, d)
	}
}
