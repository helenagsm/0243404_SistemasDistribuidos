package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

var (
	enc = binary.BigEndian
)

const (
	lenWidth = 8
)

type store struct {
	*os.File
	mu       sync.Mutex
	storebuf *bufio.Writer
	size     uint64
}

func newStore(f *os.File) (s *store, err error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	sizee := uint64(fi.Size())

	return &store{
		File:     f,
		storebuf: bufio.NewWriter(f),
		size:     sizee,
	}, err
}

func (storee *store) Append(p []byte) (n uint64, pos uint64, err error) {
	storee.mu.Lock()
	defer storee.mu.Unlock()

	pos = storee.size
	size := uint64(len(p))
	err = binary.Write(storee.storebuf, enc, size)
	if err != nil {
		return 0, 0, err
	}

	n2, err := storee.storebuf.Write(p)
	if err != nil {
		return 0, 0, err
	}
	n2 += lenWidth
	storee.size += uint64(n2)

	n = uint64(n2)

	return n, pos, nil
}

func (s *store) Read(n uint64) (p []byte, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.storebuf.Flush(); err != nil {
		return nil, err
	}

	p = make([]byte, lenWidth)
	if _, err = s.File.ReadAt(p, int64(n)); err != nil {
		return nil, err
	}

	b := make([]byte, enc.Uint64(p))

	if _, err := s.File.ReadAt(b, int64(n+lenWidth)); err != nil {
		return nil, err
	}

	return b, nil
}

func (s *store) ReadAt(p []byte, off int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.storebuf.Flush(); err != nil {
		return 0, err
	}
	return s.File.ReadAt(p, off)
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.storebuf.Flush()
	if err != nil {
		return err
	}

	err = s.File.Close()
	if err != nil {
		return err
	}

	return nil
}

func (s *store) Remove() error {
	if err := s.Close(); err != nil {
		return err
	}
	return os.Remove(s.Name())
}
