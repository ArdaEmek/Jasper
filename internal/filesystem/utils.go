package filesystem

import (
	"errors"
	"path/filepath"
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
