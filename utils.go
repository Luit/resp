package resp

func isspace(c byte) bool {
	return c == ' ' ||
		c == '\t' ||
		c == '\n' ||
		c == '\v' ||
		c == '\f' ||
		c == '\r'
}

func ishexdigit(c byte) bool {
	return ('0' <= c && c <= '9') ||
		(c >= 'a' && c <= 'f') ||
		(c >= 'A' && c <= 'F')
}

func hextobyte(c byte) byte {
	if '0' <= c && c <= '9' {
		return c - '0'
	}
	if c < 'a' {
		return 10 + (c - 'A')
	}
	return 10 + (c - 'a')
}
