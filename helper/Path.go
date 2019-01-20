package helper

import (
	"os"
	"strings"
)

func GetPathWithoutSuffix(path string) string {
	return strings.TrimRight(path, "/")
}

func GetPathJoin(path ...string) string {
	pathStr := make([]string, len(path))
	for index, item := range path {
		pathStr[index] = GetPathWithoutSuffix(item)
	}

	return strings.Join(pathStr, "/")
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}