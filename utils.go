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
	switch c {
	case '0':
		return 0
	case '1':
		return 1
	case '2':
		return 2
	case '3':
		return 3
	case '4':
		return 4
	case '5':
		return 5
	case '6':
		return 6
	case '7':
		return 7
	case '8':
		return 8
	case '9':
		return 9
	case 'A', 'a':
		return 0xa
	case 'B', 'b':
		return 0xb
	case 'C', 'c':
		return 0xc
	case 'D', 'd':
		return 0xd
	case 'E', 'e':
		return 0xe
	case 'F', 'f':
		return 0xf
	}
	return 0xff
}
