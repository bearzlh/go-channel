package helper

import (
	"fmt"
	"strconv"
)

func RoundString(value string, number int) float64 {
	floatValue, _ := strconv.ParseFloat(value, 64)
	format := fmt.Sprintf("%d", number)
	floatValue, _ = strconv.ParseFloat(fmt.Sprintf("%."+format+"f", floatValue), 64)
	return floatValue
}

func RoundFloat(value float64, number int) float64 {
	format := fmt.Sprintf("%d", number)
	value, _ = strconv.ParseFloat(fmt.Sprintf("%."+format+"f", value), 64)
	return value
}