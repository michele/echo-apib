package apib

import "net/http"

type Writer struct {
	http.ResponseWriter
	text        string
	wroteHeader bool
}

func NewWriter(res http.ResponseWriter) *Writer {
	return &Writer{
		ResponseWriter: res,
	}
}

func (aw *Writer) Write(b []byte) (int, error) {
	if !aw.wroteHeader {
		aw.WriteHeader(http.StatusOK)
	}
	aw.text = aw.text + string(b)
	return aw.ResponseWriter.Write(b)
}

func (aw *Writer) WriteHeader(code int) {
	aw.ResponseWriter.WriteHeader(code)
	if aw.wroteHeader {
		return
	}
	aw.wroteHeader = true
}

func (aw *Writer) Body() string {
	return aw.text
}
