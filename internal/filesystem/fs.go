package filesystem

import (
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
	GetFileInfo(fileName string) (os.FileInfo, error)
	ReadFile(fileName string) (*FileReader, error)
	WriteFile(fileName string) (*FileWriter, error)
}

var (
	fmInstance *fileManager = nil
	once       sync.Once
)

func New(path string) (FileManager, error) {
	err := os.MkdirAll(path, 0750)
	if err != nil {
		if os.IsExist(err) {
			log.Println("Storage not found, creating a new one.")
		} else {
			return nil, err
		}
	}

	once.Do(func() {
		fmInstance = &fileManager{
			path,
		}
	})

	log.Println("Storage initialized.")
	return fmInstance, nil
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
	path, err := fm.joinPath(fileName)
	if err != nil {
		return err
	}

	return os.Remove(path)
}

func (fm *fileManager) GetFileInfo(fileName string) (os.FileInfo, error) {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return nil, err
	}

	return os.Stat(path)
}
