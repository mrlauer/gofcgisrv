package gofcgisrv

import (
	"bytes"
	"io"
	"sync"
)

// streamReader is really sort of a piper. Maybe
type streamReader struct {
	buffer  bytes.Buffer
	lock    sync.Mutex
	gotData *sync.Cond
	err     error
}

func newStreamReader() *streamReader {
	s := new(streamReader)
	s.gotData = sync.NewCond(&s.lock)
	return s
}

func (sr *streamReader) Read(data []byte) (int, error) {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	// Wait for something to show up
	for sr.buffer.Len() == 0 && sr.err == nil {
		sr.gotData.Wait()
	}
	if sr.buffer.Len() == 0 {
		return 0, sr.err
	}
	return sr.buffer.Read(data)
}

func (sr *streamReader) Write(data []byte) (int, error) {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	if sr.err == nil {
		n, err := sr.buffer.Write(data)
		sr.gotData.Signal()
		return n, err
	}
	return 0, sr.err
}

func (sr *streamReader) Close() error {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	sr.err = io.EOF
	sr.gotData.Signal()
	return nil
}

// streamWriter writes data as FCGI records.
type streamWriter struct {
	w    io.Writer
	tp   recordType
	id   requestId
	lock sync.Mutex
}

func newStreamWriter(w io.Writer, tp recordType, id requestId) *streamWriter {
	return &streamWriter{w: w, tp: tp, id: id}
}

func (sw *streamWriter) Write(data []byte) (int, error) {
	sw.lock.Lock()
	defer sw.lock.Unlock()
	if len(data) == 0 {
		return 0, nil
	}
	rec := record{sw.tp, sw.id, data}
	err := writeRecord(sw.w, rec)
	// How much did we actually write? Just say nothing if we got an error.
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (sw *streamWriter) Close() error {
	// Close means writing an empty string
	return writeRecord(sw.w, record{sw.tp, sw.id, nil})
}
