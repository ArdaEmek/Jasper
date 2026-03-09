package filesystem

import (
	"log"
	"os"
	"sync"
)

type FileManager struct {
	workingDir string
}

var (
	fmInstance *FileManager = nil
	once       sync.Once
)

func New(path string) (*FileManager, error) {
	err := os.MkdirAll(path, 0750)
	if err != nil {
		if os.IsExist(err) {
			log.Println("Storage not found, creating a new one.")
		} else {
			return nil, err
		}
	}

	once.Do(func() {
		fmInstance = &FileManager{
			path,
		}
	})

	log.Println("Storage initialized.")
	return fmInstance, nil
}

func GetInstance() *FileManager {
	return fmInstance
}

func (fm *FileManager) GetWorkingDir() string {
	return fm.workingDir
}

func (fm *FileManager) CreateFile(fileName string) (*os.File, error) {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
}

func (fm *FileManager) GetFile(fileName string) (*os.File, error) {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(path, os.O_RDONLY, 0644)
}

func (fm *FileManager) IsExist(fileName string) (bool, error) {
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

func (fm *FileManager) DeleteFile(fileName string) error {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return err
	}

	return os.Remove(path)
}

func (fm *FileManager) GetFileInfo(fileName string) (os.FileInfo, error) {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return nil, err
	}

	return os.Stat(path)
}
