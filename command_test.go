package resp

import (
	"bytes"
	"io"
	"testing"
)

var commandTests = []struct {
	input  string
	output []string
}{
	{
		`

		test`,
		[]string{"test"},
	},
	{
		`test`,
		[]string{"test"},
	},
	{
		`test test`,
		[]string{"test", "test"},
	},
	{
		`SET somekey somevalue`,
		[]string{"SET", "somekey", "somevalue"},
	},
	{
		`SET 'some key' somevalue`,
		[]string{"SET", "some key", "somevalue"},
	},
	{
		`  	SET 'some key' 			 somevalue  `,
		[]string{"SET", "some key", "somevalue"},
	},
	{
		`SET 'some \' k\ey' somevalue`,
		[]string{"SET", `some ' k\ey`, "somevalue"},
	},
	{
		`SET "some key" somevalue`,
		[]string{"SET", "some key", "somevalue"},
	},
	{
		`"\x4a\x4B\xEF"`,
		[]string{"JK\xef"},
	},
	{
		`"test\n\t\t\t\t\"test"`,
		[]string{`test
				"test`},
	},
	{
		`this'is' a"bit" str"ange"`,
		[]string{"thisis", "abit", "strange"},
	},
	{
		`"\x01\x23\x45\x67\x89\xab\xcd\xef"`,
		[]string{"\x01\x23\x45\x67\x89\xab\xcd\xef"},
	},
	// Array of Bulk String variants
	{
		"*2\r\n$3\r\nGET\r\n$10\r\nsome value\r\n",
		[]string{"GET", "some value"},
	},
	{
		"\n\t \t\n   \n*2\r\n$3\r\nGET\r\n$10\r\nsome value\r\n",
		[]string{"GET", "some value"},
	},
}

func TestParseCommand(t *testing.T) {
	for _, test := range commandTests {
		cmd := []byte(test.input)
		if cmd[len(cmd)-1] != '\n' {
			cmd = append(cmd, '\n')
		}
		length, out, err := parseCommand(cmd)
		if err != nil {
			t.Error(err)
			continue
		}
		var output []string
		for _, b := range out {
			output = append(output, string(b))
		}
		if len(test.output) != len(output) {
			t.Errorf("expected %+#v, got %+#v", test.output, output)
			continue
		}
		for i := range test.output {
			if test.output[i] != string(output[i]) {
				t.Errorf("expected %+#v, got %+#v", test.output, output)
				break
			}
		}
		if length != len(cmd) {
			t.Errorf("expected length %d, got %d", len(test.input)+1, length)
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

func TestCommandReader(t *testing.T) {
	input := new(bytes.Buffer)
	for _, test := range commandTests {
		in := test.input
		if in[len(in)-1] != '\n' {
			in += "\n"
		}
		_, err := input.WriteString(in)
		if err != nil {
			t.Fatal(err)
		}
	}
	r := NewCommandReader(perByteReader{r: input})
	for _, test := range commandTests {
		_, out, err := r.Read()
		if err != nil {
			t.Error(err)
		}
		var output []string
		for _, b := range out {
			output = append(output, string(b))
		}
		if len(test.output) != len(output) {
			t.Errorf("expected %+#v, got %+#v", test.output, output)
			continue
		}
		for i := range test.output {
			if test.output[i] != string(output[i]) {
				t.Errorf("expected %+#v, got %+#v", test.output, output)
				break
			}
		}
	}
}

func TestCommandReaderFullRead(t *testing.T) {
	input := new(bytes.Buffer)
	for _, test := range commandTests {
		in := test.input
		if in[len(in)-1] != '\n' {
			in += "\n"
		}
		_, err := input.WriteString(in)
		if err != nil {
			t.Fatal(err)
		}
	}
	r := NewCommandReader(input)
	for _, test := range commandTests {
		_, out, err := r.Read()
		if err != nil {
			t.Error(err)
		}
		var output []string
		for _, b := range out {
			output = append(output, string(b))
		}
		if len(test.output) != len(output) {
			t.Errorf("expected %+#v, got %+#v", test.output, output)
			continue
		}
		for i := range test.output {
			if test.output[i] != string(output[i]) {
				t.Errorf("expected %+#v, got %+#v", test.output, output)
				break
			}
		}
	}
}

func TestUnbalancedQuotes(t *testing.T) {
	tests := []string{
		`SET 'some key`,
		`SET 'some 'key`,
		`SET "some key`,
		`SET "some "key`,
	}
	for _, test := range tests {
		_, _, err := parseCommand([]byte(test + "\n"))
		if err != errUnbalancedQuotes {
			t.Errorf("expected errUnbalancedQuotes, got %v", err)
		}
	}
}

func TestIncompleteCommand(t *testing.T) {
	tests := []string{
		`SET some' key' somevalue`,
		"*2\r\n$3\r\nGET\r\n$4test\r",
	}
	for _, test := range tests {
		for n := len(test); n > 0; n-- {
			_, _, err := parseCommand([]byte(test[:n]))
			if err != errIncompleteCommand {
				t.Errorf("expected errUnbalancedQuotes, got %v", err)
			}
		}
	}
}

func TestCommandErrors(t *testing.T) {
	tests := []string{
		"*\r\n",
		"*1\r\n$\r\n",
		"*1\r\n:1234\r\n",
		"*1\r\n$asdf\r\n",
		"*1\r\n$-1\r\n",
	}
	for _, test := range tests {
		_, _, err := parseCommand([]byte(test))
		if err == nil {
			t.Errorf("expected an error, got nil for %q", test)
		}
	}
}
