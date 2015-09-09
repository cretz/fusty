package model

type Execution struct {
	Device    *Device `json:"device"`
	Job       *Job    `json:"job"`
	Timestamp int64   `json:"timestamp"`
}
