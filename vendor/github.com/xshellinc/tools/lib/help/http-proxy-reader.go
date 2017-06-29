package help

import "io"

// Create new proxy reader over bar
// Takes io.Reader or io.ReadCloser
func NewHttpProxyReader(r io.Reader, f func(n int, err error)) io.Reader {
	return &HttpProxyReader{r, f}
}

// It's proxy reader, implement io.Reader
type HttpProxyReader struct {
	io.Reader
	callback func(n int, err error)
}

func (self *HttpProxyReader) Read(p []byte) (n int, err error) {
	n, err = self.Reader.Read(p)
	self.callback(n, err)

	return
}

// Close the reader when it implements io.Closer
func (self *HttpProxyReader) Close() (err error) {
	if closer, ok := self.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return
}
