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

var errIncompleteInlineCommand = errors.New("incomplete inline command")

// parseInline parses an Inline Command, returning the length of the first
// Inline Command that is present in data, and a slice parts of the
// interpreted command. The data in parts will be a (parsed) copy of whatever
// is found in data (it is owned by the caller).
func parseInline(data []byte) (length int, parts [][]byte, err error) {
	length = bytes.IndexByte(data, '\n') + 1
	if length == 0 {
		// bytes.IndexByte was -1, so no '\n' present
		err = errIncompleteInlineCommand
		return
	}
	var pos int
	// skip leading blanks
	for pos < len(data) && data[pos] != 0 && isspace(data[pos]) {
		pos++
	}
	for pos < len(data) {
		var (
			partLength int
			part       []byte
		)
		partLength, part, err = parseInlinePart(data[pos:])
		if err != nil {
			return
		}
		pos += partLength
		parts = append(parts, part)
		// skip trailing blanks
		for pos < len(data) && data[pos] != 0 && isspace(data[pos]) {
			pos++
		}
	}
	length = pos
	return
}

const errUnbalancedQuotes respError = "ERR Protocol error: unbalanced quotes in request"

// parseInlinePart parses a single Inline Command part from data, returning
// the length of the part in data and a parsed part.
func parseInlinePart(data []byte) (length int, part []byte, err error) {
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
