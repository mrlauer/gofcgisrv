package gofcgisrv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

type requestId uint16
type recordType uint8

const fcgiVersion uint8 = 1

// Request types.
const (
	fcgiBeginRequest recordType = iota + 1
	fcgiAbortRequest
	fcgiEndRequest
	fcgiParams
	fcgiStdin
	fcgiStdout
	fcgiStderr
	fcgiData
	fcgiGetValues
	fcgiGetValuesResult
	fcgiUnknown
	fcgiMaxtype = fcgiUnknown
)

// Roles
const (
	fcgiResponder uint16 = iota + 1
	fcgiAuthorizer
	fcgiFilter
)

// ProtocolStatus
const (
	fcgiRequestComplete uint8 = iota
	fcgiCanMpxConn
	fcgiOverloaded
	fcgiUnknownRole
)

// Variable names
const (
	fcgiMaxConns  = "FCGI_MAX_CONNS"
	fcgiMaxReqs   = "FCGI_MAX_REQS"
	fcgiMpxsConns = "FCGI_MPXS_CONNS"
)

var pad [255]byte

type record struct {
	Type    recordType
	Id      requestId
	Content []byte
}

func writeRecord(w io.Writer, rec record) error {
	clength := len(rec.Content)
	if clength > 0xffff {
		return errors.New("Content too large for record")
	}
	// Padding
	plength := (-uint16(clength)) & 7
	// Should we check for failure? Let's not bother for the first set, as
	// we'll always be doing this into a buffer.
	binary.Write(w, binary.BigEndian, fcgiVersion)
	binary.Write(w, binary.BigEndian, rec.Type)
	binary.Write(w, binary.BigEndian, rec.Id)
	binary.Write(w, binary.BigEndian, uint16(clength))
	binary.Write(w, binary.BigEndian, uint8(plength))
	w.Write([]byte{0})
	if _, err := w.Write(rec.Content); err != nil {
		return err
	}
	if plength != 0 {
		if _, err := w.Write(pad[:plength]); err != nil {
			return err
		}
	}
	return nil
}

func _read(r io.Reader, data interface{}) error {
	return binary.Read(r, binary.BigEndian, data)
}

func readRecord(r io.Reader) (record, error) {
	var rec record
	var version uint8
	var clength uint16
	var plength uint8
	if err := _read(r, &version); err != nil {
		return rec, err
	} else if version != fcgiVersion {
		return rec, errors.New("Unknown version")
	}
	if err := _read(r, &rec.Type); err != nil {
		return rec, err
	}
	if err := _read(r, &rec.Id); err != nil {
		return rec, err
	}
	if err := _read(r, &clength); err != nil {
		return rec, err
	}
	if err := _read(r, &plength); err != nil {
		return rec, err
	}
	// Skip one byte
	if _, err := r.Read(pad[:1]); err != nil {
		return rec, err
	}
	if clength != 0 {
		rec.Content = make([]byte, clength)
		if _, err := io.ReadFull(r, rec.Content); err != nil {
			return rec, err
		}
	}
	if plength != 0 {
		if _, err := io.ReadFull(r, pad[:plength]); err != nil {
			return rec, err
		}
	}
	return rec, nil
}

func writePairLen(w io.Writer, val int) error {
	if val <= 127 {
		return binary.Write(w, binary.BigEndian, uint8(val))
	}
	return binary.Write(w, binary.BigEndian, uint32(val)|0x80000000)
}

func readPairLen(r io.Reader) (int, error) {
	var buf [4]byte
	if _, err := r.Read(buf[:1]); err != nil {
		return 0, err
	}
	if buf[0] > 127 {
		if _, err := r.Read(buf[1:]); err != nil {
			return 0, err
		}
		buf[0] &= 0x7f
		return int(binary.BigEndian.Uint32(buf[:])), nil
	}
	return int(buf[0]), nil
}

func writeNameValue(w io.Writer, name, value string) error {
	buffer := bytes.NewBuffer(nil)
	if err := writePairLen(buffer, len(name)); err != nil {
		return err
	}
	if err := writePairLen(buffer, len(value)); err != nil {
		return err
	}
	if _, err := io.WriteString(buffer, name); err != nil {
		return err
	}
	if _, err := io.WriteString(buffer, value); err != nil {
		return err
	}
	_, err := io.Copy(w, buffer)
	return err
}

func readNameValue(r io.Reader) (name, value string, err error) {
	nameLen, err := readPairLen(r)
	if err != nil {
		return name, value, err
	}
	valueLen, err := readPairLen(r)
	if err != nil {
		return name, value, err
	}
	if nameLen > 0 {
		nameBytes := make([]byte, nameLen)
		_, err = r.Read(nameBytes)
		if err != nil {
			return
		}
		name = string(nameBytes)
	}
	if valueLen > 0 {
		valueBytes := make([]byte, valueLen)
		_, err = r.Read(valueBytes)
		if err != nil {
			return
		}
		value = string(valueBytes)
	}
	return name, value, nil
}

func writeGetValues(w io.Writer, names ...string) error {
	buffer := bytes.NewBuffer(nil)
	for _, name := range names {
		writeNameValue(buffer, name, "")
	}
	return writeRecord(w, record{fcgiGetValues, 0, buffer.Bytes()})
}

func writeBeginRequest(w io.Writer, reqId requestId, role uint16, flags byte) error {
	content := [8]byte{byte(role >> 8), byte(role & 0xff), flags, 0, 0, 0, 0, 0}
	return writeRecord(w, record{fcgiBeginRequest, reqId, content[:]})
}
