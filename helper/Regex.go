package helper

import "regexp"

//检测正则
func RegexpMatch(content string, reg string) [][]byte {
	byteContent := []byte(content)
	regex := regexp.MustCompile(reg)
	s := regex.FindSubmatch(byteContent)
	return s
}