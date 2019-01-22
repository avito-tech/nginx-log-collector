package clickhouse

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

func MakeUrl(dsn, table string) (string, error) {
	if !strings.HasSuffix(dsn, "/") {
		dsn += "/"
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return "", errors.Wrap(err, "unable to make uploader url")
	}

	q := u.Query()
	q.Set("query", fmt.Sprintf("INSERT INTO %s FORMAT JSONEachRow", table))
	q.Set("input_format_skip_unknown_fields", "1")
	u.RawQuery = q.Encode()
	return u.String(), nil
}
