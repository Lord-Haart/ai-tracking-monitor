package main

import (
	"log"
	"time"

	_agent "com.cne/ai-tracking-monitor/agent"
	_db "com.cne/ai-tracking-monitor/db"
	_rpcclient "com.cne/ai-tracking-monitor/rpcclient"
	_types "com.cne/ai-tracking-monitor/types"
	_utils "com.cne/ai-tracking-monitor/utils"
)

func doCheck() {
	now := time.Now()

	crawlerInfoList := _db.QueryAllCrawlerInfos(now)
	log.Printf("[INFO] %d active crawler(s) found\n", len(crawlerInfoList))

	findCrawlerInfo_ := func(carrierCode string) *_db.CrawlerInfoPo {
		for _, ci := range crawlerInfoList {
			if ci.CarrierCode == carrierCode {
				return ci
			}
		}

		return nil
	}

	trackingSearchList := make([]*_rpcclient.TrackingSearch, 0)

	reqTime := time.Now()
	for _, crawlerInfo := range crawlerInfoList {
		// 为每个爬虫发送一个查询请求。
		if seqNo, err := _utils.NewSeqNo(); err != nil {
			panic(err)
		} else {
			trackingSearchList = append(trackingSearchList, &_rpcclient.TrackingSearch{
				ReqTime:     reqTime,
				SeqNo:       seqNo,
				CarrierCode: crawlerInfo.CarrierCode,
				Language:    _types.LangEN,
				TrackingNo:  crawlerInfo.HeartBeatNo,
			})
		}
	}

	// 监控请求使用最高优先级。
	if keys, err := _rpcclient.PushTrackingSearchToQueue(_types.PriorityHighest, trackingSearchList); err != nil {
		// 推送查询对象到任务队列失败，放弃轮询缓存和拉取查询对象。
	} else {
		// 从缓存拉取查询对象（以及查询结果）。
		if trackingSearchList, err := _rpcclient.PullTrackingSearchFromCache(_types.PriorityHighest, keys); err != nil {
			panic(err)
		} else {
			for _, ts := range trackingSearchList {
				resultStatus := 0
				if _agent.IsSuccess(ts.AgentCode) {
					resultStatus = 1
				}

				var timing int64
				var endTime time.Time
				if !_utils.IsZeroTime(ts.AgentEndTime) {
					endTime = ts.AgentEndTime
				} else {
					endTime = time.Now()
				}

				timing = endTime.Sub(ts.ReqTime).Milliseconds()

				crawlerInfo := findCrawlerInfo_(ts.CarrierCode)
				if crawlerInfo != nil {
					log.Printf("[INFO] %s(carrier-code=%s, tracking-no=%s) checked\n", crawlerInfo.Name, crawlerInfo.CarrierCode, crawlerInfo.HeartBeatNo)

					_db.SaveHealthLog(crawlerInfo.CarrierId, ts.TrackingNo, int(timing), resultStatus, endTime)
				}
			}
		}
	}
}
