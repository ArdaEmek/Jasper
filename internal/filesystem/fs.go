package filesystem

import (
	"os"
	"sync"
)

type FileManager struct {
}

var (
	fmInstance *FileManager
	once       sync.Once
)

func New() *FileManager {
	if fmInstance != nil {
		return fmInstance
	}

	once.Do(func() {
		fmInstance = &FileManager{}
	})

	return fmInstance
}

func (fm *FileManager) CreateFile(filePath string, ch chan<- error) error {
	_, err := os.Create(filePath)

	ch <- err
	return err
}
