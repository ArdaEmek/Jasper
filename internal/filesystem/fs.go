package filesystem

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

type fileManager struct {
	workingDir string
}

type FileManager interface {
	GetWorkingDir() string
	CreateFile(fileName string) (*os.File, error)
	GetFile(fileName string) (*os.File, error)
	IsExist(fileName string) (bool, error)
	DeleteFile(fileName string) error
	DeleteFiles(fileNames []string) error
	GetFileInfo(fileName string) (os.FileInfo, error)
	ReadFile(fileName string) (*FileReader, error)
	WriteFile(fileName string) (*FileWriter, error)
	NewSegmentedReader(segs []Segment) *SegmentedReader
	ParseRange(h string, size int64) (start, end int64, ok bool)
}

var (
	fmInstance FileManager
	initErr    error
	once       sync.Once
)

func New(path string) (FileManager, error) {
	once.Do(func() {
		if path == "" {
			path = "./data"
		}
		if err := os.MkdirAll(path, 0750); err != nil {
			initErr = fmt.Errorf("create storage directory %q: %w", path, err)
			return
		}
		fmInstance = &fileManager{workingDir: path}
		log.Printf("Storage initialized at %s.", path)
	})

	return fmInstance, initErr
}

func GetInstance() FileManager {
	return fmInstance
}

func (fm *fileManager) GetWorkingDir() string {
	return fm.workingDir
}

func (fm *fileManager) CreateFile(fileName string) (*os.File, error) {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
}

func (fm *fileManager) GetFile(fileName string) (*os.File, error) {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(path, os.O_RDONLY, 0644)
}

func (fm *fileManager) IsExist(fileName string) (bool, error) {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func (fm *fileManager) DeleteFile(fileName string) error {
	if fileName == "" {
		return errors.New("empty file name")
	}
	path, err := fm.joinPath(fileName)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func (fm *fileManager) DeleteFiles(fileNames []string) error {
	var errs []error
	for _, name := range fileNames {
		if err := fm.DeleteFile(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("delete %s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

func (fm *fileManager) GetFileInfo(fileName string) (os.FileInfo, error) {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return nil, err
	}

	return os.Stat(path)
}
