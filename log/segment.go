package log

import (
	"fmt"
	"os"
	"path"

	"google.golang.org/protobuf/proto"
	api "modulo.com/proyecto_distribuido/api/v1"
)

type segment struct {
	store                  *store
	index                  *index
	baseOffset, nextOffset uint64
	config                 Config
}

func newSegment(dir string, baseOffset uint64, c Config) (*segment, error) {
	s := &segment{
		baseOffset: baseOffset,
		config:     c,
	}
	var err error
	storeFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".store")),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, err
	}
	if s.store, err = newStore(storeFile); err != nil {
		return nil, err
	}
	indexFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".index")),
		os.O_RDWR|os.O_CREATE,
		0644,
	)
	if err != nil {
		return nil, err
	}
	if s.index, err = newIndex(indexFile, c); err != nil {
		return nil, err
	}
	if off, _, err := s.index.Read(-1); err != nil {
		s.nextOffset = baseOffset
	} else {
		s.nextOffset = baseOffset + uint64(off) + 1
	}

	return s, nil
}

func (s *segment) Append(r *api.Record) (uint64, error) {
	nuevoOffset := s.nextOffset
	r.Offset = nuevoOffset
	store, err := proto.Marshal(r)
	if err != nil {
		return 0, err
	}

	_, pos, err := s.store.Append(store)
	if err != nil {
		return 0, err
	}

	if err = s.index.Write(
		// Index offsets are relative to the base offset on the store file
		uint32(s.nextOffset-uint64(s.baseOffset)),
		uint64(pos),
	); err != nil {
		return 0, err
	}

	s.nextOffset++
	return nuevoOffset, nil
}

func (s *segment) Read(off uint64) (*api.Record, error) {
	_, pos, err := s.index.Read(int64(off - s.baseOffset))
	if err != nil {
		return nil, err
	}

	data, err := s.store.Read(pos)
	if err != nil {
		return nil, err
	}
	record := &api.Record{}
	if err := proto.Unmarshal(data, record); err != nil {
		return nil, err
	}

	return record, nil
}

func (s *segment) IsMaxed() bool {
	if s.store.size >= s.config.Segment.MaxStoreBytes {
		return true
	}
	if s.index.size >= s.config.Segment.MaxIndexBytes {
		return true
	}
	return false
}

func (s *segment) Remove() error {
	if err := s.index.Remove(); err != nil {
		return err
	}
	if err := s.store.Remove(); err != nil {
		return err
	}
	return nil
}

func (s *segment) Close() error {
	err := s.store.File.Close()
	if err != nil {
		return err
	}

	err = s.index.file.Close()
	if err != nil {
		return err
	}
	return nil
}
