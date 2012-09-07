package gofcgisrv

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
)

func phpCgiHandler(w http.ResponseWriter, r *http.Request) {
	filepath := path.Join("./testdata/", r.URL.Path)
	env := []string{
		"REDIRECT_STATUS=200",
		"SCRIPT_FILENAME=" + filepath,
	}

	ServeHTTP(NewCGI("php-cgi"), env, w, r)
}

func Example_php() {
	server := httptest.NewServer(http.HandlerFunc(phpCgiHandler))
	defer server.Close()
	url := server.URL

	text := "This is a test!"
	resp, err := http.Post(url+"/echo.php", "text/plain", strings.NewReader(text))
	if err == nil {
		fmt.Printf("Status: %v\n", resp.StatusCode)
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("Response: %s\n", body)
	}

	// Output:
	// Status: 200
	// Response: This is a test!
}
