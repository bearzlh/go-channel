package helper

import (
	"fmt"
	"strings"
	"time"
)

//时间格式化
func TimeFormat(format string, timestamp int64) string {
	now := time.Unix(time.Now().Unix(), 0)
	if timestamp != 0 {
		now = time.Unix(timestamp, 0)
	}
	a := []string{}
	for _, char := range format {
		switch char {
		case 'Y':
			a = append(a, fmt.Sprintf("%d", now.Year()))
			break;
		case 'm':
			a = append(a, fmt.Sprintf("%02d", now.Month()))
			break;
		case 'd':
			a = append(a, fmt.Sprintf("%02d", now.Day()))
			break;
		case 'H':
			a = append(a, fmt.Sprintf("%02d", now.Hour()))
			break;
		case 'i':
			a = append(a, fmt.Sprintf("%02d", now.Minute()))
			break;
		case 's':
			a = append(a, fmt.Sprintf("%02d", now.Second()))
			break;
		default:
			a = append(a, string(char))
			break;
		}
	}
	return strings.Join(a, "")
}

//日期转化为时间戳
func FormatTimeStamp(format string, layout string) int64 {
	if layout == "" {
		layout = "2006-01-02 15:04:05"
	}
	loc, _ := time.LoadLocation("Local")
	theTime, _ := time.ParseInLocation(layout, format, loc)
	sr := theTime.Unix()
	return sr
}

func FormatToLayout(format string) string {
	layoutMap := map[byte]string{
		'Y': "2006",
		'm': "01",
		'd': "02",
		'H': "15",
		'i': "04",
		's': "05",
	}

	a := []string{}
	for _, char := range format {
		switch char {
		case 'Y':
			a = append(a, layoutMap[byte(char)])
			break;
		case 'm':
			a = append(a, layoutMap[byte(char)])
			break;
		case 'd':
			a = append(a, layoutMap[byte(char)])
			break;
		case 'H':
			a = append(a, layoutMap[byte(char)])
			break;
		case 'i':
			a = append(a, layoutMap[byte(char)])
			break;
		case 's':
			a = append(a, layoutMap[byte(char)])
			break;
		default:
			a = append(a, string(char))
			break;
		}
	}
	return strings.Join(a, "")
}

//格式化时间跨度
func FormatTime(second int64) string {
	duration := time.Duration(second * time.Second.Nanoseconds())
	return duration.String()
}

func GetMinDuration(layout string) int64 {
	str := layout[len(layout)-1]
	var res int64
	switch str {
	case 'd':
		res = 86400
		break;
	case 'H':
		res = 3600
		break;
	case 'i':
		res = 60
		break;
	case 's':
		res = 1
		break;
	default:
		res = 3600
	}

	return res
}