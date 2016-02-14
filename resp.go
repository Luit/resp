// resp is a package to read and interpret Redis Serialization Protocol data.
package resp // import "luit.eu/resp"

import (
	"bytes"
	"errors"
)

var (
	errInvalidInt = errors.New("invalid integer")
	errIncomplete = errors.New("incomplete")
)

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
