package resp

import (
	"bytes"
	"io"
	"testing"
)

var tests = []string{
	"+OK\r\n",
	"-ERR something\r\n",
	":-1\r\n",
	"$-1\r\n",
	"$0\r\n\r\n",
	":12345\r\n",
	"*3\r\n:1\r\n:2\r\n:3\r\n",
	"*2\r\n$3\r\nGET\r\n$4\r\ntest\r\n",
}

func TestParse(t *testing.T) {
	for _, test := range tests {
		length, err := parse([]byte(test))
		if err != nil {
			t.Errorf("%q error: %v", test, err)
		}
		if length != len(test) {
			t.Errorf("%q wrong length, expected %d, got %d", test, len(test), length)
		}
	}
}

type perByteReader struct {
	r io.Reader
}

func (r perByteReader) Read(p []byte) (n int, err error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return r.r.Read(p)
}

func TestReaderPerByte(t *testing.T) {
	readerTest(t, func(r io.Reader) io.Reader { return perByteReader{r: r} })
}

func TestReaderFullRead(t *testing.T) {
	readerTest(t, func(r io.Reader) io.Reader { return r })
}

func readerTest(t *testing.T, f func(io.Reader) io.Reader) {
	input := new(bytes.Buffer)
	for _, test := range tests {
		_, err := input.WriteString(test)
		if err != nil {
			t.Fatal(err)
		}
	}
	r := NewReader(f(input))
	for _, test := range tests {
		data, err := r.Read()
		if err != nil {
			t.Error(err)
		}
		if string(data) != test {
			t.Errorf("expected %q, got %q", test, string(data))
			continue
		}
	}
}

func TestInvalid(t *testing.T) {
	tests := []string{
		":asdf\r\n",
		":\r\n",
		":9223372036854775809\r\n",
		"hoi",
	}
	for _, test := range tests {
		_, err := parse([]byte(test))
		if err == nil {
			t.Errorf("input %q expected error, got nil", test)
		}
	}
}
