package parser

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var nginxErrorLogVariables = []string{
	", client: ",
	", server: ",
	", login: ",
	", upstream: ",
	", request: ",
	", subrequest: ",
	", host: ",
	", referrer: ",
}

func NginxErrorLogMessage(msg []byte, out map[string]interface{}) error {
	msg = bytes.TrimPrefix(msg, []byte(" "))
	if len(msg) < 19 {
		return errors.New("line too short")
	}
	// TODO check field names

	text := string(msg)

	// XXX skip timestamp parsing because of missing timezone. Use current server time instead
	// timestamp, err := time.ParseInLocation("2006/01/02 15:04:05", text[:19], time.Local)
	// if err != nil {
	// 	return err
	// }
	// out["event_datetime"] = timestamp.Format(dateTimeFmt)
	// out["event_date"] = timestamp.Format("2006-01-02")

	var p1, p2 int

	p1 = strings.IndexByte(text, '[')
	p2 = strings.IndexByte(text, ']')

	if p1 < 0 || p2 < 0 || p2 <= p1 {
		return errors.New("can't find error level")
	}

	out["level"] = text[p1+1 : p2]

	text = text[p2+2:]

	p1 = strings.IndexByte(text, '#')
	if p1 < 0 {
		return errors.New("PID not found")
	}

	var err error
	out["pid"], err = strconv.Atoi(text[:p1])
	if err != nil {
		return fmt.Errorf("wrong PID: %s", text[:p1])
	}

	text = text[p1+1:]

	p1 = strings.IndexByte(text, ':')
	if p1 < 0 {
		return errors.New("TID not found")
	}
	out["tid"], err = strconv.Atoi(text[:p1])
	if err != nil {
		return fmt.Errorf("wrong TID: %s", text[:p1])
	}

	text = text[p1+2:]

	if text[0] == '*' {
		p1 = strings.IndexByte(text, ' ')
		if p1 < 0 {
			return errors.New("SID not found")
		}
		out["sid"], err = strconv.Atoi(text[1:p1])
		if err != nil {
			return fmt.Errorf("wrong SID: %s", text[1:p1])
		}

		text = text[p1+1:]
	}

	indexes := make([]int, 0, len(nginxErrorLogVariables)+1)

	for i := 0; i < len(nginxErrorLogVariables); i++ {
		p1 = strings.LastIndex(text, nginxErrorLogVariables[i])
		if p1 < 0 {
			continue
		}
		indexes = append(indexes, p1)
	}

	if len(indexes) == 0 {
		out["message"] = text
		return nil
	}

	indexes = append(indexes, len(text))

	sort.Ints(indexes)

	out["message"] = text[:indexes[0]]

	for i := 0; i < len(indexes)-1; i++ {
		s := text[indexes[i]:indexes[i+1]]
		p1 = strings.IndexByte(s, ':')
		v := s[p1+2:]
		if len(v) > 0 && v[0] == '"' {
			v = v[1:]
		}
		if len(v) > 0 && v[len(v)-1] == '"' {
			v = v[:len(v)-1]
		}
		k := s[2:p1]
		if k == "server" {
			k = "server_name"
		} else if k == "referrer" {
			k = "http_referer"
		}
		out[k] = v
	}

	return nil
}
