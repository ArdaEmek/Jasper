package filesystem

import (
	"bufio"
	"os"
)

type FileReader struct {
	File   *os.File
	Reader *bufio.Reader
	closed bool
}

func (fr *FileReader) Read(p []byte) (n int, err error) {
	return fr.Reader.Read(p)
}

func (fr *FileReader) Close() error {
	if fr.closed {
		return nil
	}
	fr.closed = true
	return fr.File.Close()
}

func (fm *fileManager) ReadFile(fileName string) (*FileReader, error) {
	file, err := fm.GetFile(fileName)
	if err != nil {
		return nil, err
	}

	return &FileReader{
		File:   file,
		Reader: bufio.NewReader(file),
	}, nil
}

type FileWriter struct {
	File   *os.File
	Writer *bufio.Writer
	closed bool
}

func (fw *FileWriter) Write(p []byte) (n int, err error) {
	return fw.Writer.Write(p)
}

func (fw *FileWriter) Close() error {
	if fw.closed {
		return nil
	}
	fw.closed = true

	if err := fw.Writer.Flush(); err != nil {
		fw.File.Close()
		return err
	}

	return fw.File.Close()
}

func (fm *fileManager) WriteFile(fileName string) (*FileWriter, error) {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	writer := bufio.NewWriter(file)
	return &FileWriter{File: file, Writer: writer}, nil
}
