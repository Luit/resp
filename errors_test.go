package resp

import "testing"

func TestParseError(t *testing.T) {
	tests := []struct {
		input  string
		output string
		prefix string
	}{
		{"-\r\n", "", ""},
		{"-ERR some error\r\n", "ERR some error", "ERR"},
	}
	for _, test := range tests {
		output := ParseError([]byte(test.input))
		if output.Error() != test.output {
			t.Errorf("output expected %q, got %q", test.output, output)
		}
		if output.Prefix() != test.prefix {
			t.Errorf("prefix expected %q, got %q", test.prefix, output.Prefix())
		}
		input := output.RESP()
		if string(input) != test.input {
			t.Errorf("output expected %q, got %q", test.input, string(input))
		}
	}
}
