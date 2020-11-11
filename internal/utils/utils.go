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
