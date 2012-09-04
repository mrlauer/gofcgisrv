package gofcgisrv

import (
	"bufio"
	"io"
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

	appendEnv("QUERY_STRING", r.URL.RawQuery)
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

// ProcessResponse adds any returned header data to the response header and sends the rest
// to the response body.
func ProcessResponse(stdout io.Reader, w http.ResponseWriter, r *http.Request) {
	bufReader := bufio.NewReader(stdout)
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
