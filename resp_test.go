package resp

import "testing"

func TestParseInline(t *testing.T) {
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
	}
	for _, test := range tests {
		_, out, err := parseInline([]byte(test.input + "\n"))
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
	}
}

func TestUnbalancedQuotes(t *testing.T) {
	tests := []string{
		`SET 'some key`,
		`SET "some key`,
	}
	for _, test := range tests {
		_, _, err := parseInline([]byte(test + "\n"))
		if err != errUnbalancedQuotes {
			t.Errorf("expected errUnbalancedQuotes, got %v", err)
		}
	}
}
