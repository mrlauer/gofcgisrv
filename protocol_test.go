package gofcgisrv

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestWriteRecord(t *testing.T) {
	data := []struct {
		rec            record
		expectedHeader []byte
	}{
		{
			record{
				Type:    37,
				Id:      4321,
				Content: []byte("This is some content."),
			},
			[]byte{1, 37, 16, 225, 0, 21, 3, 0},
		},
		{
			record{
				Type:    3,
				Id:      1,
				Content: []byte("01234567"),
			},
			[]byte{1, 3, 0, 1, 0, 8, 0, 0},
		},
		{
			record{
				Type:    3,
				Id:      1,
				Content: nil,
			},
			[]byte{1, 3, 0, 1, 0, 0, 0, 0},
		},
	}
	for _, d := range data {
		rec := d.rec
		expectedHeader := d.expectedHeader
		buf := bytes.NewBuffer(nil)
		err := writeRecord(buf, rec)
		b := buf.Bytes()
		if err != nil {
			t.Errorf("Error writing record %v", err)
		}
		if !bytes.Equal(b[:8], expectedHeader) {
			t.Errorf("Header was %v", b[:8])
		}
		if !bytes.Equal(b[8:len(rec.Content)+8], []byte(rec.Content)) {
			t.Errorf("Content was %q", b[8:len(rec.Content)+8])
		}
		if len(b)%8 != 0 {
			t.Errorf("Length was %d", len(b))
		}
	}
}

func TestReadRecord(t *testing.T) {
	data := []struct {
		str      string
		expected record
	}{
		{
			"\x01\x25\x10\xe1\x00\x15\x03\x00This is some content.\000\000\000",
			record{
				Type:    37,
				Id:      4321,
				Content: []byte("This is some content."),
			},
		},
		{
			"\x01\x03\x00\x01\x00\x08\x00\x0001234567",
			record{
				Type:    3,
				Id:      1,
				Content: []byte("01234567"),
			},
		},
	}
	for i, d := range data {
		reader := strings.NewReader(d.str)
		rec, err := readRecord(reader)
		if err != nil {
			t.Errorf("Error writing record %v", err)
		}
		if rec.Type != d.expected.Type {
			t.Errorf("type %d vs %d at %d\n", rec.Type, d.expected.Type, i)
		}
		if rec.Id != d.expected.Id {
			t.Errorf("id %d vs %d at %d\n", rec.Id, d.expected.Id, i)
		}
		if !bytes.Equal(rec.Content, d.expected.Content) {
			t.Errorf("content %s vs %s at %d\n", rec.Content, d.expected.Content, i)
		}
		b := make([]byte, 16)
		if n, err := reader.Read(b); n != 0 || err != io.EOF {
			t.Errorf("Reader %d was not at eof; read %s, %v", i, b[:n], err)
		}
	}
}

var longString0 string = strings.Repeat("aXq", 100)
var longString1 string = strings.Repeat("poop", 102)

func TestWriteValues(t *testing.T) {
	data := []struct{ name, value, expected string }{
		{"Foo", "Bar", "\003\003FooBar"},
		{longString0, "Bar", "\x80\000\001\x2c\003" + longString0 + "Bar"},
		{longString0, longString1, "\x80\000\001\x2c\x80\000\001\x98" + longString0 + longString1},
		{"Foo", longString0, "\x03\x80\000\001\x2c" + "Foo" + longString0},
		{"Foo", "", "\003\000Foo"},
	}
	for i, d := range data {
		buffer := bytes.NewBuffer(nil)
		writeNameValue(buffer, d.name, d.value)
		s := string(buffer.Bytes())
		if s != d.expected {
			t.Errorf("Got %q, not %q, at %d", s, d.expected, i)
		}
		reader := bytes.NewBufferString(s)
		n, v, err := readNameValue(reader)
		if err != nil || n != d.name || v != d.value {
			t.Errorf("Get %s = %s with %v at %d", n, v, err, i)
		}
	}
}
