package common

func Substr(input string, start int, length int) string {
	inputRunes := []rune(input)

	if start >= len(inputRunes) {
		return ""
	}

	if start+length > len(inputRunes) {
		length = len(inputRunes) - start
	}

	return string(inputRunes[start : start+length])
}

func SubstrAll(input string, start int) string {
	inputRunes := []rune(input)

	if start >= len(inputRunes) {
		return ""
	}

	return string(inputRunes[start:])
}
