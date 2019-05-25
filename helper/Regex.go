package helper

import "regexp"

//检测正则
func RegexpMatch(content string, reg string) []string {
	regex := regexp.MustCompile(reg)
	return regex.FindStringSubmatch(content)
}