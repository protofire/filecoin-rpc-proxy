package utils

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"
)

func FileExists(name string) bool {
	stat, err := os.Stat(name)
	return !os.IsNotExist(err) && !stat.IsDir()
}

func GetUserHome() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
}

func Equal(i interface{}, j interface{}) bool {

	convert := func(i interface{}) interface{} {
		switch i := i.(type) {
		case float64:
			return int(i)
		case float32:
			return int(i)
		case int:
			return i
		case int8:
			return int(i)
		case int16:
			return int(i)
		case int32:
			return int(i)
		case int64:
			return int(i)
		case byte:
			return int(i)
		}
		return i
	}

	return convert(i) == convert(j)
}

func Min(v1 int, v2 ...int) int {
	min := v1
	for _, v := range v2 {
		if v < min {
			min = v
		}
	}
	return min
}

func Read(r io.ReadCloser) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	defer func() {
		if err := r.Close(); err != nil {
			logger.Log.Errorf("cannot close http request body: %v", err)
		}
	}()

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	return body, nil
}
