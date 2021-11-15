package db

import (
	"time"
)

const (
	insertCrawlerHealthLog string = `insert into crawler_health_log (crawler_id, tracking_no, timing, result_status, create_time, update_time, status, crawler_resp_body, result_note) 
	values(?, ?, ?, ?, ?, ?, ?, ?, ?)`
)

func SaveHealthLog(carrierId int64, trackingNo string, timing int, resultStatus int, datePoint time.Time, crawlerRespBody, resultNote string) int64 {
	if result, err := db.Exec(insertCrawlerHealthLog, carrierId, trackingNo, timing, resultStatus, datePoint, datePoint, 1 /*status*/, crawlerRespBody, resultNote); err != nil {
		panic(err)
	} else {
		if lastRowId, err := result.LastInsertId(); err != nil {
			panic(err)
		} else {
			return lastRowId
		}
	}
}
