package clickhouse

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const TIMEOUT = time.Minute * 5

func Upload(uploadUrl string, data []byte) error {
	return UploadReader(uploadUrl, bytes.NewReader(data))
}

func UploadReader(uploadUrl string, data io.Reader) error {
	req, err := http.NewRequest("POST", uploadUrl, data)
	client := &http.Client{Timeout: TIMEOUT}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "upload http error")
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("clickhouse response status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
