package resp

// Error is an error value that can be directly used as a RESP Error value to
// a client
type Error interface {
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
