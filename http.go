package gofcgisrv

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
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
func ProcessResponse(stdoutRead io.Reader, rw http.ResponseWriter, req *http.Request) {
	linebody := bufio.NewReaderSize(stdoutRead, 1024)
	headers := make(http.Header)
	statusCode := 0
	for {
		line, isPrefix, err := linebody.ReadLine()
		if isPrefix {
			rw.WriteHeader(http.StatusInternalServerError)
			logger.Printf("fcgi: long header line from subprocess.")
			return
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			logger.Printf("fcgi: error reading headers: %v", err)
			return
		}
		if len(line) == 0 {
			break
		}
		parts := strings.SplitN(string(line), ":", 2)
		if len(parts) < 2 {
			logger.Printf("fcgi: bogus header line: %s", string(line))
			continue
		}
		header, val := parts[0], parts[1]
		header = strings.TrimSpace(header)
		val = strings.TrimSpace(val)
		switch {
		case header == "Status":
			if len(val) < 3 {
				logger.Printf("fcgi: bogus status (short): %q", val)
				return
			}
			code, err := strconv.Atoi(val[0:3])
			if err != nil {
				logger.Printf("fcgi: bogus status: %q", val)
				logger.Printf("fcgi: line was %q", line)
				return
			}
			statusCode = code
		default:
			headers.Add(header, val)
		}
	}

	/* // TODO : handle internal redirect ?
	if loc := headers.Get("Location"); loc != "" {
		if strings.HasPrefix(loc, "/") && h.PathLocationHandler != nil {
			h.handleInternalRedirect(rw, req, loc)
			return
		}
		if statusCode == 0 {
			statusCode = http.StatusFound
		}
	}
	*/

	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	// Copy headers to rw's headers, after we've decided not to
	// go into handleInternalRedirect, which won't want its rw
	// headers to have been touched.
	for k, vv := range headers {
		for _, v := range vv {
			rw.Header().Add(k, v)
		}
	}

	rw.WriteHeader(statusCode)

	_, err := io.Copy(rw, linebody)
	if err != nil {
		logger.Printf("cgi: copy error: %v", err)
	}
}
