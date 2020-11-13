package utils

import (
	"os"
	"os/user"
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
