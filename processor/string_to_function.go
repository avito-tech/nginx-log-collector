package processor

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var regLimitMaxLength = regexp.MustCompile(`^limitMaxLength\((\d+)\)$`)

func tryLimitMaxLength(fName string) (TransformFunc, error) {
	m := regLimitMaxLength.FindStringSubmatch(fName)
	if len(m) == 2 {
		arg0, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, errors.Wrap(err, "unable to find function argument")
		}
		return func(s string) []byte {
			return limitMaxLength(s, arg0)
		}, err
	}
	return nil, fmt.Errorf("unable to parse limitMaxLength; name=%s", fName)
}

func strToFn(fName string) (TransformFunc, error) {
	if fName == "ipToUint32" {
		return ipToUint32, nil
	} else if fName == "toArray" {
		return toArray, nil
	} else if strings.HasPrefix(fName, "limitMaxLength") {
		return tryLimitMaxLength(fName)
	}
	return nil, fmt.Errorf("unknown function: %s", fName)
}
