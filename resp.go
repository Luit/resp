// Package resp is a package to read and interpret Redis Serialization
// Protocol data.
package resp // import "luit.eu/resp"

import (
	"bytes"
	"errors"
	"io"
)

var (
	errInvalid    = errors.New("invalid resp")
	errInvalidInt = errors.New("invalid integer")
	errIncomplete = errors.New("incomplete")
)

// Reader is a reader you can use to read from a socket in a way similar to
// what the Redis client does when listening for responses from a server.
type Reader struct {
	r   io.Reader
	buf *bytes.Buffer
}

// NewReader creates a new Reader from any io.Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r:   r,
		buf: new(bytes.Buffer),
	}
}

// Read reads a single command, and returns it in parts. The slices in return
// values data and parts are only valid until the next call to Read.
func (r *Reader) Read() (data []byte, err error) {
	var length int
	for err == nil {
		length, err = parse(r.buf.Bytes())
		if err == errIncomplete {
			err = readmore(r.r, r.buf)
		} else if err == nil {
			break
		}
	}
	data = r.buf.Next(length)
	return
}

// read into writer, recovering from bytes.ErrTooLarge panic on write
func readmore(r io.Reader, buf io.Writer) (err error) {
	defer func() {
		if perr := recover(); perr != nil {
			if perr == bytes.ErrTooLarge {
				err = bytes.ErrTooLarge
			} else {
				panic(perr)
			}
		}
	}()
	p := make([]byte, 1024)
	var n int
	n, err = r.Read(p)
	if err != nil {
		return
	}
	_, err = buf.Write(p[:n])
	return
}

func parse(data []byte) (length int, err error) {
	if len(data) < 1 {
		err = errIncomplete
		return
	}
datatype:
	switch data[0] {
	case '+', '-':
		length, _, err = parseLine(data[1:])
	case '$':
		length, _, err = parseBulkString(data[1:])
	case ':':
		length, _, err = parseInteger(data[1:])
	case '*':
		var narg int
		length, narg, err = parseInteger(data[1:])
		if err != nil {
			break
		}
		length++ // for ease of use
		for n := 0; n < narg; n++ {
			var partlen int
			partlen, err = parse(data[length:])
			length += partlen
			if err != nil {
				length--
				break datatype
			}
		}
		length-- // for ease of use
	default:
		err = errInvalid
	}
	if err != nil {
		return
	}
	length++
	return
}

// parseInteger parses an integer + "\r\n" from data, returning the length of
// bytes parsed from data and the parsed integer.
func parseInteger(data []byte) (length int, n int, err error) {
	length = bytes.IndexByte(data, '\r') + 2
	if length == 1 || len(data) < length {
		// bytes.IndexByte was -1, so no \n present, or \r was found,
		// but not enough data to include \n
		length = 0
		err = errIncomplete
		return
	}
	if length == 2 {
		// just "\r\n", which is a bad int
		err = errInvalidInt
		return
	}
	var (
		prev int
		pos  int
	)
	for pos < length-2 {
		if !('0' <= data[pos] && data[pos] <= '9') {
			err = errInvalidInt
			return
		}
		n *= 10
		n += int(data[pos] - '0')
		if n < prev {
			// integer overflow
			err = errInvalidInt
			return
		}
		pos++
	}
	return
}

func parseBulkString(data []byte) (length int, part []byte, err error) {
	var (
		n int
	)
	length, n, err = parseInteger(data)
	// TODO: find out definite limits of stuff like this
	if err == errInvalidInt || n > 512*1024*1024 {
		err = errInvalidBulkLength
	}
	if err != nil {
		return
	}
	if len(data) < length+n+2 {
		err = errIncomplete
		return
	}
	part = data[length : length+n]
	length += n + 2
	return
}

func parseLine(data []byte) (length int, part []byte, err error) {
	length = bytes.IndexByte(data, '\r') + 2
	if length == 1 || len(data) < length {
		// bytes.IndexByte was -1, so no \n present, or \r was found,
		// but not enough data to include \n
		length = 0
		err = errIncomplete
		return
	}
	part = data[:length-2]
	return
}
