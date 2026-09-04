package common

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type streamingOnlyBodyStorage struct {
	*bytes.Reader
}

func (s *streamingOnlyBodyStorage) Close() error { return nil }
func (s *streamingOnlyBodyStorage) Size() int64  { return s.Reader.Size() }
func (s *streamingOnlyBodyStorage) IsDisk() bool { return true }
func (s *streamingOnlyBodyStorage) Bytes() ([]byte, error) {
	return nil, errors.New("multipart parser must not materialize disk-backed storage")
}

type reopenableTestReader struct {
	*bytes.Reader
}

func (r *reopenableTestReader) Close() error { return nil }
func (r *reopenableTestReader) NewReader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("replayed"))), nil
}

func TestReaderOnlyPreservesReplayWithoutExposingClose(t *testing.T) {
	original := &reopenableTestReader{Reader: bytes.NewReader([]byte("original"))}
	reader := ReaderOnly(original)

	_, exposesClose := reader.(io.Closer)
	require.False(t, exposesClose)

	replayable, ok := reader.(interface {
		NewReader() (io.ReadCloser, error)
	})
	require.True(t, ok, "non-closing wrapper must preserve request body replay")

	replayed, err := replayable.NewReader()
	require.NoError(t, err)
	defer replayed.Close()
	data, err := io.ReadAll(replayed)
	require.NoError(t, err)
	require.Equal(t, []byte("replayed"), data)
}

func TestParseMultipartFormReusableStreamsBodyStorage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var encoded bytes.Buffer
	writer := multipart.NewWriter(&encoded)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	part, err := writer.CreateFormFile("image", "input.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	storage := &streamingOnlyBodyStorage{Reader: bytes.NewReader(encoded.Bytes())}
	c.Set(KeyBodyStorage, storage)

	form, err := ParseMultipartFormReusable(c)
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2", form.Value["model"][0])
	require.Len(t, form.File["image"], 1)

	position, err := storage.Seek(0, io.SeekCurrent)
	require.NoError(t, err)
	require.Zero(t, position, "body storage must be rewound after multipart parsing")
}
