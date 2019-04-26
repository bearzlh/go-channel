package object

import "workerChannel/helper"

type Doc struct {
	Index Index `json:"index"`
	Content MsgInterface `json:"content"`
}

type Index struct {
	IndexName IndexContent `json:"index"`
}

type IndexContent struct {
	Index string `json:"_index"`
	Type string `json:"_type"`
}

type MsgInterface interface {
	GetTimestamp() int64
	GetPickTime() string
	GetLogType() string
	GetLogLine() int64
	GetIndex(string, string, int64) string
	GetIndexObj(string, string, int64) Index
	GetName() string
	GetJobId() string
}

func GetIndex(env string, format string, time int64, flag string) string {
	index := flag + "-" + helper.TimeFormat(format, time)
	if env != "" {
		index = env + "-" + index
	}

	return index
}