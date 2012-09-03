package gofcgisrv

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

func TestStreamReader(t *testing.T) {
	sr := newStreamReader()
	io.WriteString(sr, "abc")
	data := make([]byte, 1024)
	n, err := sr.Read(data)
	if err != nil {
		t.Errorf("Read error %v", err)
	}
	if string(data[:n]) != "abc" {
		t.Errorf("Read %s, not %s", data[:n], "abc")
	}

	ch := make(chan string)
	go func() {
		n, err := sr.Read(data)
		if err != nil {
			t.Error(err)
		}
		ch <- string(data[:n])
	}()
	// Let the goroutine execute a bit
	time.Sleep(time.Millisecond)
	go func() {
		sr.Write([]byte("ABCD"))
	}()
	str := <-ch
	if str != "ABCD" {
		t.Errorf("Read %s, not %s", str, "ABCD")
	}

	ss := []string{"foo", "bar", strings.Repeat("xyzå", 100)}
	for _, s := range ss {
		sr.Write([]byte(s))
	}
	go sr.Close()
	ssj := strings.Join(ss, "")
	read, err := ioutil.ReadAll(sr)
	if err != nil || string(read) != ssj {
		t.Errorf("Read %s with error %v", read, err)
	}
}

func TestStreamWriter(t *testing.T) {
	var buffer bytes.Buffer
	sw := newStreamWriter(&buffer, fcgiStdout, 3)
	io.WriteString(sw, "Foo!")
	io.WriteString(sw, "This is data")
	io.WriteString(sw, "\000\001abc")
	sw.Close()

	expected := "\001\006\000\003\000\004\004\000Foo!\000\000\000\000" +
		"\001\006\000\003\000\x0c\004\000This is data\000\000\000\000" +
		"\001\006\000\003\000\x05\003\000\000\001abc\000\000\000" +
		"\001\006\000\003\000\x00\000\000"

	str := string(buffer.Bytes())
	if str != expected {
		t.Errorf("Got\n%q\nnot\n%q", str, expected)
	}
}
