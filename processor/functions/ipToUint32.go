package functions

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"strings"
)

type ipToUint32 struct{}

func (f *ipToUint32) Call(ip string) FunctionResult {
	b := bytes.Buffer{}
	result := FunctionPartialResult{}

	b.WriteByte('"')
	if strings.Contains(ip, ":") { // XXX ignore ipv6 for now
		b.WriteByte('0')
	} else if parsed := net.ParseIP(ip); parsed == nil {
		b.WriteByte('0')
	} else {
		b.WriteString(strconv.FormatUint(uint64(binary.BigEndian.Uint32(parsed[12:16])), 10))
	}
	b.WriteByte('"')

	result.Value = b.Bytes()
	return FunctionResult{result}
}
