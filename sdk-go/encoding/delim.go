package encoding

import (
	"bufio"
	"bytes"
)

func DelimSplitFunc(delim string) bufio.SplitFunc {
	delimBytes := []byte(delim)
	delimLen := len(delimBytes)

	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.Index(data, delimBytes); i >= 0 {
			// We have a full token.
			token := data[0:i]
			return i + delimLen, token, nil
		}

		// There is no delimiter, so we need to read more data.
		if atEOF {
			return len(data), data, nil
		}

		return 0, nil, nil
	}
}
