package gofcgisrv

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"strings"
)

// HTTPEnv sets up an environment with standard HTTP/CGI variables.
func HTTPEnv(start []string, r *http.Request) []string {
	env := make([]string, 0, 10)
	env = append(env, start...)

	appendEnv := func(key, value string) {
		env = append(env, key+"="+value)
	}

	appendEnv("REQUEST_METHOD", r.Method)
	appendEnv("SERVER_PROTOCOL", "HTTP/1.1")
	appendEnv("GATEWAY_INTERFACE", "CGI/1.1")
	appendEnv("REQUEST_URI", r.URL.String())

	host, port, err := net.SplitHostPort(r.Host)
	if err != nil {
		host, port = r.Host, "80"
	}
	appendEnv("SERVER_NAME", host)
	appendEnv("SERVER_PORT", port)

	if len(r.URL.RawQuery) > 0 {
		appendEnv("QUERY_STRING", r.URL.RawQuery)
	}
	if t := r.Header.Get("Content-type"); t != "" {
		appendEnv("CONTENT_TYPE", t)
	}
	if l := r.Header.Get("Content-length"); l != "" {
		appendEnv("CONTENT_LENGTH", l)
	}

	for key := range r.Header {
		upper := strings.ToUpper(key)
		cgikey := "HTTP_" + strings.Replace(upper, "-", "_", 1)
		appendEnv(cgikey, r.Header.Get(key))
	}
	return env
}

func ServeHTTP(s *Server, env []string, w http.ResponseWriter, r *http.Request) {
	env = HTTPEnv(env, r)

	outreader, outwriter := io.Pipe()
	stderr := bytes.NewBuffer(nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer outwriter.Close()
		err := s.Request(env, r.Body, outwriter, stderr)
		if err != nil {
			// There should not be anything in stdout. We should really guard against that.
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()

	// Add any headers produced by the application, and skip to the response.
	ProcessResponse(outreader, w, r)
	<-done
}

// ProcessResponse adds any returned header data to the response header and sends the rest
// to the response body.
func ProcessResponse(stdout io.Reader, w http.ResponseWriter, r *http.Request) {
	bufReader := bufio.NewReader(stdout)
	mimeReader := textproto.NewReader(bufReader)
	hdr, err := mimeReader.ReadMIMEHeader()
	if err != nil {
		// We got nothing! Assume there is an error. Should be more robust.
		return
	}
	if err == nil {
		for k, vals := range hdr {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
	}
	statusCode := http.StatusOK
	if status := hdr.Get("Status"); status != "" {
		delete(w.Header(), "Status")
		// Parse the status code
		var code int
		if n, _ := fmt.Sscanf(status, "%d", &code); n == 1 {
			statusCode = int(code)
		}
	}
	// Are there other fields we need to rewrite? Probably!
	w.WriteHeader(statusCode)
	io.Copy(w, bufReader)
}
