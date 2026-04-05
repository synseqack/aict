package meta

import (
	"time"
)

type TimeInfo struct {
	Unix int64 `xml:"modified,attr"`
	AgoS int64 `xml:"modified_ago_s,attr"`
}

func Now() int64 {
	return time.Now().Unix()
}

func AgoSeconds(unixTime int64) int64 {
	elapsed := time.Now().Unix() - unixTime
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

func TimeInfoFrom(unixTime int64) TimeInfo {
	return TimeInfo{
		Unix: unixTime,
		AgoS: AgoSeconds(unixTime),
	}
}

func FormatTime(unixTime int64) string {
	return time.Unix(unixTime, 0).UTC().Format("2006-01-02 15:04:05")
}
