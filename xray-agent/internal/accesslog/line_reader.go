package accesslog

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

var errLineTooLong = errors.New("line exceeds configured limit")

// readBoundedLine retains at most maxBytes and reports whether it reached a newline.
func readBoundedLine(reader *bufio.Reader, maxBytes int) ([]byte, int64, bool, error) {
	if maxBytes <= 0 {
		return nil, 0, false, fmt.Errorf("line size limit must be positive")
	}
	line := make([]byte, 0, min(maxBytes, 4096))
	var consumed int64
	for {
		fragment, readErr := reader.ReadSlice('\n')
		consumed += int64(len(fragment))
		remaining := maxBytes - len(line)
		if len(fragment) > remaining {
			if remaining > 0 {
				line = append(line, fragment[:remaining]...)
			}
			switch {
			case readErr == nil:
				return nil, consumed, true, errLineTooLong
			case errors.Is(readErr, bufio.ErrBufferFull):
				return nil, consumed, false, errLineTooLong
			case errors.Is(readErr, io.EOF):
				return nil, consumed, false, errors.Join(errLineTooLong, io.EOF)
			default:
				return nil, consumed, false, readErr
			}
		}
		line = append(line, fragment...)

		switch {
		case readErr == nil:
			return line, consumed, true, nil
		case errors.Is(readErr, bufio.ErrBufferFull):
			continue
		case errors.Is(readErr, io.EOF):
			if consumed == 0 {
				return nil, 0, false, io.EOF
			}
			return line, consumed, false, io.EOF
		default:
			return nil, consumed, false, readErr
		}
	}
}
