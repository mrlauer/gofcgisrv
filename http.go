package gofcgisrv

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"strings"
)

func parseEnv(envStr string) (key, value string, err error) {
	if idx := strings.Index(envStr, "="); idx > 0 {
		return envStr[:idx], envStr[idx+1:], nil
	}
	return "", "", errors.New("Not a valid environment string.")
}

// HTTPEnv sets up an environment with standard HTTP/CGI variables.
func HTTPEnv(start []string, r *http.Request) []string {
	envMap := make(map[string]string)
	env := make([]string, 0, 10)
	for _, e := range start {
		if k, v, err := parseEnv(e); err == nil {
			envMap[k] = v
			env = append(env, e)
		}
	}

	appendEnv := func(key, value string) {
		if _, ok := envMap[key]; !ok {
			env = append(env, key+"="+value)
		}
	}

	appendEnv("SCRIPT_NAME", "")
	appendEnv("REQUEST_METHOD", r.Method)
	appendEnv("SERVER_PROTOCOL", "HTTP/1.1")
	appendEnv("GATEWAY_INTERFACE", "CGI/1.1")
	appendEnv("REQUEST_URI", r.URL.String())
	appendEnv("HTTP_HOST=", r.Host)

	host, port, err := net.SplitHostPort(r.Host)
	if err != nil {
		host, port = r.Host, "80"
	}
	appendEnv("REMOTE_ADDR", r.RemoteAddr)
	appendEnv("SERVER_NAME", host)
	appendEnv("SERVER_PORT", port)

	appendEnv("QUERY_STRING", r.URL.RawQuery)
	if t := r.Header.Get("Content-type"); t != "" {
		appendEnv("CONTENT_TYPE", t)
	}

	for key := range r.Header {
		upper := strings.ToUpper(key)
		cgikey := "HTTP_" + strings.Replace(upper, "-", "_", -1)
		appendEnv(cgikey, r.Header.Get(key))
	}
	return env
}

// ServeHTTP serves an http request using FastCGI
func ServeHTTP(s Requester, env []string, w http.ResponseWriter, r *http.Request) {
	env = HTTPEnv(env, r)

	var body io.Reader = r.Body
	// CONTENT_LENGTH is special and important
	if l := r.Header.Get("Content-length"); l != "" {
		env = append(env, "CONTENT_LENGTH="+l)
	} else if r.Body != nil {
		// Some different transfer-encoding, presumably.
		// Read the body into a buffer
		// If there is an error we'll just ignore it for now.
		buf := bytes.NewBuffer(nil)
		io.Copy(buf, r.Body)
		r.Body.Close()
		body = buf
		env = append(env, fmt.Sprintf("CONTENT_LENGTH=%d", buf.Len()))
	} else {
		env = append(env, "CONTENT_LENGTH=0")
	}

	outreader, outwriter := io.Pipe()
	stderr := bytes.NewBuffer(nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer outwriter.Close()
		err := s.Request(env, body, outwriter, stderr)
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
func ProcessResponse(stdout io.Reader, w http.ResponseWriter, r *http.Request) error {
	bufReader := bufio.NewReader(stdout)
	mimeReader := textproto.NewReader(bufReader)
	hdr, err := mimeReader.ReadMIMEHeader()
	if err != nil {
		// We got nothing! Assume there is an error. Should be more robust.
		return err
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
	return nil
}
