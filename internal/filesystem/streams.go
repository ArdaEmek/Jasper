package filesystem

import (
	"bufio"
	"io"
	"os"
)

const writeBufferSize = 256 * 1024

type FileReader struct {
	File   *os.File
	closed bool
}

func (fr *FileReader) Read(p []byte) (int, error) {
	return fr.File.Read(p)
}

func (fr *FileReader) ReadAt(p []byte, off int64) (int, error) {
	return fr.File.ReadAt(p, off)
}

func (fr *FileReader) Seek(offset int64, whence int) (int64, error) {
	return fr.File.Seek(offset, whence)
}

func (fr *FileReader) CopyTo(w io.Writer, n int64) (int64, error) {
	return io.CopyN(w, fr.File, n)
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
	return &FileReader{File: file}, nil
}

type FileWriter struct {
	File   *os.File
	Writer *bufio.Writer

	path   string
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

	if err := fw.Writer.Flush(); err != nil { // Manually abort upload if flushing fails
		_ = fw.File.Close()
		_ = os.Remove(fw.path)
		return err
	}

	return fw.File.Close()
}

func (fw *FileWriter) Abort() error {
	if fw.closed {
		return nil
	}
	fw.closed = true
	_ = fw.File.Close()
	return os.Remove(fw.path)
}

func (fm *fileManager) WriteFile(fileName string) (*FileWriter, error) {
	path, err := fm.joinPath(fileName)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		return nil, err
	}

	writer := bufio.NewWriterSize(file, writeBufferSize)
	return &FileWriter{File: file, Writer: writer, path: path}, nil
}
