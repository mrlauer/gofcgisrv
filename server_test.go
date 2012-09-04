package gofcgisrv

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
)

func serve(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "FCGI!\n")
	logger.Printf("Copying\n")
	if r.Body != nil {
		io.Copy(w, r.Body)
		r.Body.Close()
	}
	logger.Printf("Copied\n")
}

func startFCGIApp(t *testing.T, addr string) (net.Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	go fcgi.Serve(l, http.HandlerFunc(serve))
	return l, nil
}

func handleWithFCGI(s *Server, w http.ResponseWriter, r *http.Request) {
	env := make([]string, 0, 1)
	env = append(env, "REQUEST_METHOD="+r.Method)
	env = append(env, "SERVER_PROTOCOL=HTTP/1.1")
	env = append(env, "GATEWAY_INTERFACE=CGI/1.1")
	env = append(env, fmt.Sprintf("REQUEST_URI=%s", r.URL))
	buffer := bytes.NewBuffer(nil)
	s.Request(env, r.Body, buffer, buffer)

	// Add any headers produced by php, and skip to the response.
	bufReader := bufio.NewReader(buffer)
	mimeReader := textproto.NewReader(bufReader)
	hdr, err := mimeReader.ReadMIMEHeader()
	if err == nil {
		for k, vals := range hdr {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
	}
	io.Copy(w, bufReader)
}

func TestFCGI(t *testing.T) {
	addr := "127.0.0.1:9000"
	l, err := startFCGIApp(t, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// Now start an http server.
	s := NewServer(addr)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleWithFCGI(s, w, r)
	})
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
