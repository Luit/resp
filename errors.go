package resp

import "strings"

// Error is an error value that can be directly used as a RESP Error value to
// a client
type Error string

// Error makes the Error an error.
func (e Error) Error() string {
	return string(e)
}

// RESP returns the RESP-representation of this Error.
func (e Error) RESP() []byte {
	return append(append([]byte{'-'}, string(e)...), "\r\n"...)
}

// Prefix returns the RESP Error Prefix.
func (e Error) Prefix() string {
	n := strings.IndexByte(string(e), ' ')
	if n == -1 {
		return string(e)
	}
	return string(e[:n])
}

// ParseError takes a byteslice returned from Reader.Read starting with '-',
// and turns it into an Error. It blindly strips the last two and the first
// byte, assuming a correct RESP Error is passed.
func ParseError(data []byte) Error {
	if len(data) <= 3 {
		return ""
	}
	return Error(string(data[1 : len(data)-2]))
}

const (
	errUnbalancedQuotes       Error = "ERR Protocol error: unbalanced quotes in request"
	errInvalidMultibulkLength Error = "ERR Protocol error: invalid multibulk length"
	errInvalidBulkLength      Error = "ERR Protocol error: invalid bulk length"
)
