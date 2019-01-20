package service

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"
	"workerChannel/helper"
)

func ImportFile() {
	if Cf.ImportFile != "" {
		for _, item := range Cf.ReadPath {
			if item.Type == "php" {
				PositionToImport := strings.Split(Cf.ImportFile, "-")
				PositionStart := Cf.ImportFile
				PositionEnd := helper.TimeFormat(item.TimeFormat, time.Now().Unix()-3600)
				if len(PositionToImport) == 2 {
					PositionStart = PositionToImport[0]
					PositionEnd = PositionToImport[1]
				}
				L.Debug("文件导入开始", LEVEL_DEBUG)
				timeStart := helper.FormatTimeStamp(PositionStart, "200601/02_15")
				for ; ; timeStart += 3600 {
					if helper.TimeFormat(item.TimeFormat, timeStart) == PositionEnd {
						L.Debug("导入结束", LEVEL_DEBUG)
						break
					}
					log := GetLogFile(item, timeStart)
					file, err := os.Open(log)
					if err != nil {
						L.Debug(log+" not exists", LEVEL_ERROR)
						continue
					} else {
						L.Debug("导入开始"+log, LEVEL_DEBUG)
					}
					rd := bufio.NewReader(file)
					var currentId string
					fileLen := 0
					for {
						line, err := rd.ReadString('\n')
						fileLen += len(line)
						if err != nil || io.EOF == err {
							L.Debug("导入结束"+log, LEVEL_DEBUG)
							break
						}
						An.LineCount++
						currentId = PhpLineToJob(line, item.Type, currentId)
					}
				}
			}
		}
	}
}