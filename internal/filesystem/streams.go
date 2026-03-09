package filesystem

import (
	"bufio"
	"os"
)

type FileReader struct {
	File    *os.File
	Scanner *bufio.Scanner
}

func (fr *FileReader) Close() error {
	return fr.File.Close()
}

func (fm *FileManager) ReadFile(fileName string) (*FileReader, error) {
	file, err := fm.GetFile(fileName)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	return &FileReader{file, scanner}, nil
}

type FileWriter struct {
	File   *os.File
	Writer *bufio.Writer
}

func (fw *FileWriter) Close() error {
	if err := fw.Writer.Flush(); err != nil {
		return err
	}

	return fw.File.Close()
}

func (fm *FileManager) WriteFile(fileName string) (*FileWriter, error) {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	writer := bufio.NewWriter(file)
	return &FileWriter{file, writer}, nil
}
