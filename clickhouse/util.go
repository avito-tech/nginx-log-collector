package clickhouse

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func MakeUrl(dsn, table string, skipUnknownFields bool, allowErrorRatio int) (string, error) {
	if !strings.HasSuffix(dsn, "/") {
		dsn += "/"
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return "", errors.Wrap(err, "unable to make uploader url")
	}

	q := u.Query()
	q.Set("query", fmt.Sprintf("INSERT INTO %s FORMAT JSONEachRow", table))
	if skipUnknownFields {
		q.Set("input_format_skip_unknown_fields", "1")
	}
	if allowErrorRatio > 0 {
		q.Set("input_format_allow_errors_ratio", strconv.Itoa(allowErrorRatio))
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}
