package resp

import (
	"bytes"
	"errors"
	"strconv"
)


var (
	ErrIncomplete = errors.New("incomplete resp data")
	ErrInvalid = errors.New("invalid resp format")
)


func Parse(buf []byte) ([][]byte, int, error){
	if len(buf) == 0{
		return nil, 0, ErrIncomplete
	}
	if buf[0] != '*' {
		return nil,0, ErrInvalid
	}

	crlf := bytes.Index(buf, []byte("\r\n"))
	if crlf == -1{
		return nil, 0, ErrIncomplete
	}
	numElements, err := strconv.Atoi(string(buf[1:crlf]))

	if err != nil || numElements < 0{
		return nil, 0, ErrInvalid
	}
	offset := crlf + 2
	args := make([][]byte, 0, numElements)

	for i:=0; i< numElements; i++{
		if offset >= len(buf){
			return nil, 0, ErrIncomplete
		}

		if buf[offset] != '$'{
			return nil, 0, ErrInvalid
		}
		crlf = bytes.Index(buf[offset:], []byte("\r\n"))
		if crlf == -1 {
			return nil, 0, ErrIncomplete
		}
		crlf += offset
		strLen, err := strconv.Atoi(string(buf[offset+1 : crlf]))
		if err != nil || strLen < 0 {
			return nil, 0, ErrInvalid
		}
		startStr := crlf + 2
		endStr := startStr + strLen
		if endStr+2 > len(buf) {
			return nil, 0, ErrIncomplete
		}
		if buf[endStr] != '\r' || buf[endStr+1] != '\n' {
			return nil, 0, ErrInvalid
		}
		args = append(args, buf[startStr:endStr])
		offset = endStr + 2
	}
return args, offset, nil

}
