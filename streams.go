package gofcgisrv

import (
	"bytes"
	"io"
	"sync"
)

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
