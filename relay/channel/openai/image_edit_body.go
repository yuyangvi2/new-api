package openai

import (
	"io"
	"os"
	"sync"
)

type imageEditBody struct {
	file      *os.File
	path      string
	size      int64
	closeOnce sync.Once
	closeErr  error
}

func (b *imageEditBody) Read(p []byte) (int, error) {
	return b.file.Read(p)
}

func (b *imageEditBody) Size() int64 {
	return b.size
}

func (b *imageEditBody) NewReader() (io.ReadCloser, error) {
	return os.Open(b.path)
}

func (b *imageEditBody) Close() error {
	b.closeOnce.Do(func() {
		closeErr := b.file.Close()
		removeErr := os.Remove(b.path)
		if closeErr != nil {
			b.closeErr = closeErr
		} else if removeErr != nil && !os.IsNotExist(removeErr) {
			b.closeErr = removeErr
		}
	})
	return b.closeErr
}

func newImageEditBody() (*imageEditBody, error) {
	file, err := os.CreateTemp("", "new-api-image-edit-*")
	if err != nil {
		return nil, err
	}
	return &imageEditBody{file: file, path: file.Name()}, nil
}

func (b *imageEditBody) finish() error {
	info, err := b.file.Stat()
	if err != nil {
		return err
	}
	if _, err := b.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	b.size = info.Size()
	return nil
}
