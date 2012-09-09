package gofcgisrv

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
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

func headerRequester(env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	re := regexp.MustCompile(`^(\w+)=(.*)`)
	envMap := make(map[string]string)
	for _, e := range env {
		matches := re.FindStringSubmatch(e)
		if len(matches) == 3 {
			k, v := matches[1], matches[2]
			envMap[k] = v
		}
	}
	js, err := json.Marshal(envMap)
	if err != nil {
		return err
	}
	io.WriteString(stdout, "Status: 200 OK\r\n")
	io.WriteString(stdout, "Content-Type: application/json;charset=UTF-8\r\n")
	fmt.Fprintf(stdout, "Content-Length: %d\n", len(js))
	io.WriteString(stdout, "\r\n")
	stdout.Write(js)
	return nil
}

func makeHandler(f Requester, env []string) http.Handler {
	serve := func(w http.ResponseWriter, r *http.Request) {
		env := HTTPEnv(env, r)
		ServeHTTP(f, env, w, r)
	}
	return http.HandlerFunc(serve)
}

type httpTestData struct {
	name     string
	url      string
	env      []string
	f        Requester
	body     io.Reader
	status   int
	expected string
}

func testRequester(t *testing.T, data httpTestData) {
	server := httptest.NewServer(makeHandler(data.f, data.env))
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
			f:        RequesterFunc(echoRequester),
			body:     strings.NewReader("This is a test"),
			status:   200,
			expected: "This is a test",
		},
		{
			name:     "echo (different reader)",
			f:        RequesterFunc(echoRequester),
			body:     bufio.NewReader(strings.NewReader("This is a test")),
			status:   200,
			expected: "This is a test",
		},
		{
			name:     "broken",
			f:        RequesterFunc(brokenRequester),
			body:     strings.NewReader("This is a test"),
			status:   500,
			expected: _brokenReqError + "\n",
		},
	}
	for _, d := range data {
		testRequester(t, d)
	}
}

func TestHeaders(t *testing.T) {
	server := httptest.NewServer(makeHandler(RequesterFunc(headerRequester), []string{"FOO=BAR"}))
	defer server.Close()

	requrl := server.URL + "/?foo=bar%20baz"
	parsedUrl, _ := url.Parse(requrl)
	host, port, _ := net.SplitHostPort(parsedUrl.Host)

	resp, err := http.Get(requrl)
	if err != nil {
		t.Errorf("Error in get: %v", err)
		return
	}
	if resp.StatusCode != 200 {
		t.Errorf("status was %d, not %d", resp.StatusCode, 200)
	}
	var envMap map[string]string
	err = json.NewDecoder(resp.Body).Decode(&envMap)
	if err != nil {
		t.Errorf("decode error: %v\n", err)
	}
	data := []struct{ key, value string }{
		{"FOO", "BAR"},
		{"SERVER_PROTOCOL", "HTTP/1.1"},
		{"GATEWAY_INTERFACE", "CGI/1.1"},
		{"CONTENT_LENGTH", "0"},
		{"SERVER_NAME", host},
		{"SERVER_PORT", port},
		{"QUERY_STRING", "foo=bar%20baz"},
	}
	for _, d := range data {
		v := envMap[d.key]
		if v != d.value {
			t.Errorf("env[%s] was %s, not %s", d.key, v, d.value)
		}
	}
	if raddr := envMap["REMOTE_ADDR"]; !strings.HasPrefix(raddr, "127.0.0.1") {
		t.Errorf("REMOTE_ADDR was %s\n", raddr)
	}
}
