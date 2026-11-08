package filesystem

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
)

func (fm *fileManager) sanitizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	baseAbsPath, err := filepath.Abs(fm.workingDir)
	if err != nil {
		return "", err
	}

	absPath = filepath.Clean(absPath)
	baseAbsPath = filepath.Clean(baseAbsPath)

	// Checking if path goes outside the working directory
	if !strings.HasPrefix(absPath, baseAbsPath+string(filepath.Separator)) && absPath != baseAbsPath {
		return "", errors.New("path escapes base directory")
	}

	return absPath, nil
}

func (fm *fileManager) joinPath(filename string) (string, error) {
	var path string
	if filepath.IsAbs(filename) {
		path = filename
	} else {
		path = filepath.Join(fm.workingDir, filename)
	}

	return fm.sanitizePath(path)
}

func (fm *fileManager) ParseRange(h string, size int64) (start, end int64, ok bool) {
	const prefix = "bytes="
	if !strings.HasPrefix(h, prefix) || strings.Contains(h, ",") {
		return 0, 0, false
	}
	a, b, found := strings.Cut(h[len(prefix):], "-")
	if !found {
		return 0, 0, false
	}

	if a == "" { // suffix: the last N bytes
		n, err := strconv.ParseInt(b, 10, 64)
		if err != nil || n <= 0 {
			return 0, 0, false
		}
		if n > size {
			n = size
		}
		return size - n, size - 1, true
	}

	s, err := strconv.ParseInt(a, 10, 64)
	if err != nil || s < 0 {
		return 0, 0, false
	}
	e := size - 1
	if b != "" {
		e, err = strconv.ParseInt(b, 10, 64)
		if err != nil || e < s {
			return 0, 0, false
		}
		if e > size-1 {
			e = size - 1
		}
	}
	return s, e, true
}
