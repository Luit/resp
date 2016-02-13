// resp is a package to read and interpret Redis Serialization Protocol data.
package resp // import "luit.eu/resp"

import (
	"bytes"
	"errors"
)

// RESPError is an error value that can be directly used as a RESP Error value
// to a client
type RESPError interface {
	RESP() []byte
	error
}

type respError string

func (e respError) Error() string {
	return string(e)
}

func (e respError) RESP() []byte {
	return append(append([]byte{'-'}, string(e)...), "\r\n"...)
}

const (
	errUnbalancedQuotes       respError = "ERR Protocol error: unbalanced quotes in request"
	errInvalidMultibulkLength respError = "ERR Protocol error: invalid multibulk length"
	errInvalidBulkLength      respError = "ERR Protocol error: invalid bulk length"
)

var errIncompleteCommand = errors.New("incomplete command")

// parseCommand parses a Redis Serialisation Protocol Command, returning the
// length of the first Command that is present in data, and a slice parts of
// the interpreted command. The slices in slice parts can be subslices of
// slice data.
func parseCommand(data []byte) (length int, parts [][]byte, err error) {
	if len(data) < 1 {
		err = errIncompleteCommand
		return
	}
	if data[0] != '*' {
		return parseInlineCommand(data)
	}
	length += 1
	var (
		narg   int
		arglen int
	)
	arglen, narg, err = parseInteger(data[length:])
	if err == errInvalidInt || narg > 1024*1024 {
		err = errInvalidMultibulkLength
	}
	if err != nil {
		return
	}
	length += arglen
	for n := 0; n < narg; n++ {
		var part []byte
		arglen, part, err = parseCommandPart(data[length:])
		if err != nil {
			return
		}
		length += arglen
		parts = append(parts, part)
	}
	return
}

func parseCommandPart(data []byte) (length int, part []byte, err error) {
	if len(data) < 1 {
		err = errIncompleteCommand
		return
	}
	if data[0] != '$' {
		c := data[0]
		if c == '\r' || c == '\n' {
			// prevent newlines in error; invalid in protocol
			c = ' '
		}
		err = respError("ERR Protocol error: expected '$', got '" + string(c) + "'")
		return
	}
	length += 1
	var (
		intlen int
		n      int
	)
	intlen, n, err = parseInteger(data[1:])
	if err == errInvalidInt || n > 512*1024*1024 {
		err = errInvalidBulkLength
	}
	if err != nil {
		return
	}
	length += intlen
	if len(data) < length+n+2 {
		err = errIncompleteCommand
		return
	}
	part = data[length : length+n]
	length += n + 2
	return
}

var errInvalidInt = errors.New("invalid integer")

// parseInteger parses an integer + "\r\n" from data, returning the length of
// bytes parsed from data and the parsed integer.
func parseInteger(data []byte) (length int, n int, err error) {
	length = bytes.IndexByte(data, '\r') + 2
	if length == 1 || len(data) < length {
		// bytes.IndexByte was -1, so no \n present, or \r was found,
		// but not enough data to include \n
		length = 0
		err = errIncompleteCommand
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

// The data in parts will be a (parsed) copy of whatever is found in data (it
// is owned by the caller).
func parseInlineCommand(data []byte) (length int, parts [][]byte, err error) {
	length = bytes.IndexByte(data, '\n') + 1
	if length == 0 {
		// bytes.IndexByte was -1, so no '\n' present
		err = errIncompleteCommand
		return
	}
	var pos int
	// skip leading blanks
	for pos < length && data[pos] != 0 && isspace(data[pos]) {
		pos++
	}
	for pos < length {
		var (
			partLength int
			part       []byte
		)
		partLength, part, err = parseInlineCommandPart(data[pos:])
		if err != nil {
			return
		}
		pos += partLength
		parts = append(parts, part)
		// skip trailing blanks
		for pos < length && data[pos] != 0 && isspace(data[pos]) {
			pos++
		}
	}
	return
}

// parseInlineCommandPart parses a single Inline Command part from data,
// returning the length of the part in data and a parsed part.
func parseInlineCommandPart(data []byte) (length int, part []byte, err error) {
	var (
		pos  int
		done bool
		inq  bool // inside "quotes"
		insq bool // inside 'single quotes'
	)
	for pos < len(data) && !done {
		switch {
		case inq:
			switch data[pos] {
			case '\\':
				if len(data) > pos+3 &&
					data[pos+1] == 'x' &&
					ishexdigit(data[pos+2]) &&
					ishexdigit(data[pos+3]) {
					part = append(part, hextobyte(data[pos+2])<<4|
						hextobyte(data[pos+3]))
					pos += 3
				} else if len(data) > pos+1 {
					pos++
					var c byte
					switch data[pos] {
					case 'n':
						c = '\n'
					case 'r':
						c = '\r'
					case 't':
						c = '\t'
					case 'b':
						c = '\b'
					case 'a':
						c = '\a'
					default:
						c = data[pos]
					}
					part = append(part, c)
				}
			case '"':
				if len(data) > pos+1 && !isspace(data[pos+1]) {
					err = errUnbalancedQuotes
					return
				}
				inq = false
			case 0:
				err = errUnbalancedQuotes
				return
			default:
				part = append(part, data[pos])
			}
		case insq:
			switch data[pos] {
			case '\\':
				if len(data) > pos+1 && data[pos+1] == '\'' {
					pos++
					part = append(part, '\'')
				} else {
					part = append(part, data[pos])
				}
			case '\'':
				if len(data) > pos+1 && !isspace(data[pos+1]) {
					err = errUnbalancedQuotes
					return
				}
				insq = false
			default:
				part = append(part, data[pos])
			}
		default:
			switch data[pos] {
			case ' ', '\n', '\r', '\t', 0:
				done = true
			case '"':
				inq = true
			case '\'':
				insq = true
			default:
				part = append(part, data[pos])
			}
		}
		pos++
	}
	if inq || insq {
		err = errUnbalancedQuotes
		return
	}
	length = pos
	return
}
