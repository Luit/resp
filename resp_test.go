package resp

import "testing"

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input  string
		output []string
	}{
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
	}
	for _, test := range tests {
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
