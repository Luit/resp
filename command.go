package resp

import (
	"bytes"
	"io"
)

// CommandReader is a reader you can use to read from a socket in a way
// similar to what the Redis server does.
type CommandReader struct {
	r   io.Reader
	buf *bytes.Buffer
}

// NewCommandReader creates a new CommandReader from any io.Reader.
func NewCommandReader(r io.Reader) *CommandReader {
	return &CommandReader{
		r:   r,
		buf: new(bytes.Buffer),
	}
}

// Read reads a single command, and returns it in parts. The slices in return
// values data and parts are only valid until the next call to Read.
func (r *CommandReader) Read() (data []byte, parts [][]byte, err error) {
	var length int
	for err == nil {
		length, parts, err = parseCommand(r.buf.Bytes())
		if err == errIncomplete {
			err = r.readmore()
		} else if err == nil {
			break
		}
	}
	data = r.buf.Next(length)
	return
}

func (r *CommandReader) readmore() (err error) {
	defer func() {
		if perr := recover(); perr != nil {
			if perr == bytes.ErrTooLarge {
				err = bytes.ErrTooLarge
			} else {
				panic(perr)
			}
		}
	}()
	buf := make([]byte, 1024)
	var n int
	n, err = r.r.Read(buf)
	if err != nil {
		return
	}
	_, err = r.buf.Write(buf[:n])
	return
}

// parseCommand parses a Redis Serialisation Protocol Command, returning the
// length of the first Command that is present in data, and a slice parts of
// the interpreted command. The slices in slice parts can be subslices of
// slice data.
func parseCommand(data []byte) (length int, parts [][]byte, err error) {
	if len(data) < 1 {
		err = errIncomplete
		return
	}
	if data[0] != '*' {
		var pos int
		for pos < len(data) {
			newline := bytes.IndexByte(data[pos:], '\n')
			if newline == -1 {
				err = errIncomplete
				return
			}
			blank := true
			for _, c := range data[pos : pos+newline] {
				if !isspace(c) {
					blank = false
					break
				}
			}
			if !blank {
				length = pos + newline + 1
				break
			}
			pos += newline + 1
			if pos >= len(data) {
				err = errIncomplete
				return
			}
		}
		if pos != 0 {
			length, parts, err = parseCommand(data[pos:])
			length += pos
			return
		}
		parts, err = parseInlineCommand(data[:length])
		return
	}
	length++
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
		err = errIncomplete
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
	length, part, err = parseBulkString(data[1:])
	length++ // the '$' in front
	return
}

// parseInlineCommand takes exactly one line (with the final byte in data
// '\n'), and interprets it as an inline command.
func parseInlineCommand(data []byte) (parts [][]byte, err error) {
	var pos int
	// skip leading blanks
	for pos < len(data)-1 && data[pos] != 0 && isspace(data[pos]) {
		pos++
	}
	for pos < len(data)-1 {
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
		for pos < len(data) && data[pos] != 0 && isspace(data[pos]) {
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
