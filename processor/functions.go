package processor

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"strings"
)

const MIDDLE = "<...>"

func ipToUint32(ip string) []byte {
	if strings.Contains(ip, ":") { // XXX ignore ipv6 for now
		return []byte(`"0"`)
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return []byte(`"0"`)
	}
	b := bytes.Buffer{}
	b.WriteByte('"')
	b.WriteString(strconv.FormatUint(uint64(binary.BigEndian.Uint32(parsed[12:16])), 10))
	b.WriteByte('"')
	return b.Bytes()
}

func limitMaxLength(s string, maxLen int) []byte {
	// len(MIDDLE) = 5
	b := bytes.Buffer{}
	b.WriteByte('"')
	if maxLen < 5 || len(s) <= maxLen {
		b.WriteString(s)
		b.WriteByte('"')
		return b.Bytes()
	}
	rawMaxLen := maxLen - 5
	leftLen := rawMaxLen / 2
	rightLen := rawMaxLen - leftLen
	left := s[:leftLen]
	right := s[len(s)-rightLen:]
	b.WriteString(left)
	b.WriteString(MIDDLE)
	b.WriteString(right)
	b.WriteByte('"')
	return b.Bytes()
}

func toArray(s string) []byte {
	b := bytes.Buffer{}
	b.WriteByte('[')

	needComma := false
	for _, n := range strings.Split(s, " ") {
		if n != "" {
			if needComma {
				b.WriteByte(',')
			}
			if _, err := strconv.ParseFloat(n, 32); err == nil {
				b.WriteString(n)
				needComma = true
			}
		}
	}
	b.WriteByte(']')
	return b.Bytes()
}
