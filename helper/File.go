package helper

import (
	"io"
	"os"
)

func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil && fileInfo != nil && fileInfo.IsDir() {
		return true
	}
	return false
}

func IsFile(path string) bool {
	fileInfo, err := os.Stat(path)
	if err == nil && fileInfo != nil && !fileInfo.IsDir() {
		return true
	}
	return false
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

//文件写入
func FilePutContents(fileName string, content string, append bool) error {
	flag := os.O_RDWR | os.O_CREATE
	if append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	file, err := os.OpenFile(fileName, flag, 0666)

	if err != nil {
		return err
	}

	_, errWrite := io.WriteString(file, content)
	if errWrite != nil {
		return errWrite
	}

	errClose := file.Close()
	return errClose
}