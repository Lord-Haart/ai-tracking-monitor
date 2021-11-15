package db

import (
	"time"
)

const (
	insertCrawlerHealthLog string = `insert into crawler_health_log (crawler_id, tracking_no, timing, result_status, create_time, update_time, status, crawler_resp_body, result_note) 
	values(?, ?, ?, ?, ?, ?, ?, ?, ?)`

	countHealthLogByResultStatus = `select crawler_id, result_status, count(1) from crawler_health_log where create_time > ? group by crawler_id, result_status`
)

type CrawlerHealthLogRec struct {
	Id           int64
	CountOfOk    int
	CountOfError int
}

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

func CountHealthLogByResultStatus(datePoint time.Time) []*CrawlerHealthLogRec {
	if result, err := db.Query(countHealthLogByResultStatus, datePoint); err != nil {
		panic(err)
	} else {
		mr := make(map[int64]*CrawlerHealthLogRec)
		var crawlerId int64
		var resultStatus int
		var count int
		for result.Next() {
			if err := result.Scan(&crawlerId, &resultStatus, &count); err != nil {
				panic(err)
			} else {
				rr := mr[crawlerId]
				if rr == nil {
					rr = &CrawlerHealthLogRec{Id: crawlerId, CountOfOk: 0, CountOfError: 0}
				}
				if resultStatus == 0 {
					rr.CountOfOk += count
				} else {
					rr.CountOfError += count
				}
				mr[crawlerId] = rr
			}
		}

		r := make([]*CrawlerHealthLogRec, 0, len(mr))
		for _, v := range mr {
			r = append(r, v)
		}
		return r
	}
}
