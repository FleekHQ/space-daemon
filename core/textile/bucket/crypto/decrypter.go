package crypto

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"errors"
	"io"
	"os"

	"github.com/odeke-em/go-utils/tmpfile"
)

const _16KB = 16 * 1024

var DecryptErr = errors.New("message corrupt or incorrect keys")

// NewDecryptReader creates an io.ReadCloser wrapping an io.Reader using the keys and iv
// to decode the content using AES and verify HMAC.
func NewDecryptReader(r io.Reader, aesKey, iv, hmacKey []byte) (io.ReadCloser, error) {
	return newDecryptReader(r, aesKey, iv, hmacKey)
}

func newDecryptReader(r io.Reader, aesKey []byte, iv []byte, hmacKey []byte) (io.ReadCloser, error) {
	mac := make([]byte, hmacSize)
	h := hmac.New(hashFunc, hmacKey)
	dst, err := tmpfile.New(&tmpfile.Context{
		Dir:    os.TempDir(),
		Suffix: "space-encrypted-",
	})
	if err != nil {
		return nil, err
	}
	// If there is an error, try to delete the temp file.
	defer func() {
		if err != nil {
			_ = dst.Done()
		}
	}()
	b, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	d := &decryptReader{
		tmpFile: dst,
		sReader: &cipher.StreamReader{R: dst, S: cipher.NewCTR(b, iv)},
	}
	w := io.MultiWriter(h, dst)
	buf := bufio.NewReaderSize(r, _16KB)
	for {
		b, err := buf.Peek(_16KB)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF {
			left := buf.Buffered()
			if left < hmacSize {
				return nil, DecryptErr
			}
			copy(mac, b[left-hmacSize:left])
			_, err = io.CopyN(w, buf, int64(left-hmacSize))
			if err != nil {
				return nil, err
			}
			break
		}
		_, err = io.CopyN(w, buf, _16KB-hmacSize)
		if err != nil {
			return nil, err
		}
	}

	if !hmac.Equal(mac, h.Sum(nil)) {
		return nil, DecryptErr
	}

	if _, err = dst.Seek(0, 0); err != nil {
		return nil, err
	}
	return d, nil
}

// decryptReader wraps a io.Reader decrypting its content.
type decryptReader struct {
	tmpFile *tmpfile.TmpFile
	sReader *cipher.StreamReader
}

// Read implements io.Reader.
func (d *decryptReader) Read(dst []byte) (int, error) {
	return d.sReader.Read(dst)
}

// Close implements io.Closer.
func (d *decryptReader) Close() error {
	return d.tmpFile.Done()
}
