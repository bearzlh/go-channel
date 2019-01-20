package object

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
	GetIndex(int64) string
	GetIndexObj(int64) Index
	GetName() string
}