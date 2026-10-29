package telegram

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type OffsetStore interface {
	Load() (int64, error)
	Save(int64) error
}

type FileOffsetStore struct {
	Path string
}

func (s FileOffsetStore) Load() (int64, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	value, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("telegram: invalid offset file: %w", err)
	}
	return value, nil
}

func (s FileOffsetStore) Save(offset int64) error {
	dir := filepath.Dir(s.Path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	tmp, err := os.CreateTemp(dir, ".telegram-offset-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.WriteString(strconv.FormatInt(offset, 10)); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	return os.Rename(name, s.Path)
}
