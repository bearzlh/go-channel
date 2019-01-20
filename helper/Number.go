package helper

import (
	"fmt"
	"strconv"
)

func Round(value string, number int) float64 {
	floatValue, _ := strconv.ParseFloat(value, 64)
	format := fmt.Sprintf("%d", number)
	floatValue, _ = strconv.ParseFloat(fmt.Sprintf("%."+format+"f", floatValue), 64)
	return floatValue
}