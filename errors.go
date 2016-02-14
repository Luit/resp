package resp

// Error is an error value that can be directly used as a RESP Error value to
// a client
type Error string

func (e Error) Error() string {
	return string(e)
}

func (e Error) RESP() []byte {
	return append(append([]byte{'-'}, string(e)...), "\r\n"...)
}

const (
	errUnbalancedQuotes       Error = "ERR Protocol error: unbalanced quotes in request"
	errInvalidMultibulkLength Error = "ERR Protocol error: invalid multibulk length"
	errInvalidBulkLength      Error = "ERR Protocol error: invalid bulk length"
)
