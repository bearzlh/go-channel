package object

import (
	"time"
)

type SystemAnalysis struct {
	LineCount int64   `json:"line_count"`
	SleepTime float64 `json:"sleep_time"`

	//请求统计
	JobCount      int64 `json:"job_count"`
	JobQueue      int   `json:"job_queue"`
	JobProcessing int64 `json:"job_processing"`
	JobSuccess    int64 `json:"job_success"`
	BuckCount     int64 `json:"buck_count"`
	PostCurrent   int   `json:"post_current"`
	IpCurrent     int   `json:"ip_current"`

	//运行时间
	TimeStart      int64  `json:"time_start"`
	TimeStartStr   string `json:"time_start_str"`
	TimeEnd        int64  `json:"time_end"`
	TimeEndStr     string `json:"time_end_str"`
	TimeWork       string `json:"time_work"`
	TimeDelay      int64  `json:"time_delay"`
	TimeDelayStr   string `json:"time_delay_str"`
	TimePostEnd    int64  `json:"time_post_end"`
	TimePostEndStr string `json:"time_post_end_str"`

	HeapMemoryUsed uint64  `json:"memory_used_M"`
	SysMemoryUsed  uint64  `json:"sys_memory_used_M"`
	CpuRate        float64 `json:"cpu_rate"`
	Load           float64 `json:"load"`
	MemRate        float64 `json:"mem_rate"`

	CodeError    float64 `json:"code_error"`
	CodeAlert    float64 `json:"code_alert"`
	CodeCritical float64 `json:"code_critical"`

	BatchLength int `json:"batch_length"`

	HostHealth bool `json:"host_health"`
}

type WorkerMsg struct {
	SystemAnalysis

	Date      int64   `json:"date"`
	AppId     string  `json:"appid"`
	HostName  string  `json:"hostname"`

	//workJobs
	HostHealth bool `json:"host_health"`
}

func (p WorkerMsg) GetTimestamp() int64 {
	return time.Now().Unix()
}

func (p WorkerMsg) GetName() string {
	return "worker"
}

func (p WorkerMsg) GetPickTime() string {
	return ""
}

func (p WorkerMsg) GetLogType() string {
	return ""
}

//获取当前索引对象
func (p WorkerMsg) GetIndexObj(env string, format string, time int64) Index {
	return Index{IndexName: IndexContent{GetIndex(env, format, time, "worker"), "go"}}
}

//获取索引
func (p WorkerMsg) GetIndex(env string, format string, time int64) string {
	return GetIndex(env, format, time, "worker") + "/go"
}

func (p WorkerMsg) GetLogLine() int64 {
	return 0
}

func (p WorkerMsg)GetJobId() string {
	return p.TimePostEndStr
}