package helper

import (
	"os"
)

func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil && fileInfo != nil && fileInfo.IsDir() {
		return true
	}
	return false;
}

func IsFile(path string) bool {
	fileInfo, err := os.Stat(path)
	if err == nil && fileInfo != nil && !fileInfo.IsDir() {
		return true
	}
	return false;
}

func Mkdir(path string) error {
	if IsDir(path) {
		return nil
	}

	if IsFile(path) {
		err := os.Remove(path)
		if err != nil {
			return err
		}
	}

	err1 := os.MkdirAll(path, 0755)
	if err1 != nil {
		return err1
	}

	return nil
}