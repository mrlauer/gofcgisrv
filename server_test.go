package gofcgisrv

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "FCGI!\n")
	if r.Body != nil {
		io.Copy(w, r.Body)
		r.Body.Close()
	}
}

func startFCGIApp(t *testing.T, addr string) (net.Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	go fcgi.Serve(l, http.HandlerFunc(serve))
	return l, nil
}

func TestFCGI(t *testing.T) {
	addr := "127.0.0.1:9000"
	l, err := startFCGIApp(t, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// Now start an http server.
	s := NewFCGI("tcp", addr)
	http.Handle("/", s)
	server := httptest.NewServer(nil)
	defer server.Close()
	url := server.URL

	resp, err := http.Post(url+"/", "text/plain", strings.NewReader("This is a string!\n"))
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Response had status code %d\n", resp.StatusCode)
	}
	buffer := bytes.NewBuffer(nil)
	io.Copy(buffer, resp.Body)
	resp.Body.Close()
	body := string(buffer.Bytes())
	if body != "FCGI!\nThis is a string!\n" {
		t.Errorf("Response was %s\n", body)
	}
}
