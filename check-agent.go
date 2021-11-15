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
			// fmt.Printf("%v\n", seqNo)
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
				// 注意：此处规则和接口查询不同，result_status=0表示成功；result_status=1表示失败！！！！
				resultStatus := 1
				resultNote := ""
				if ts.AgentCode == _agent.AcSuccess || ts.AgentCode == _agent.AcSuccess2 {
					resultStatus = 0
				} else if ts.AgentCode == _agent.AcNoTracking {
					resultStatus = 0
					resultNote = "未查询到单号"
				} else if ts.AgentCode == _agent.AcParseFailed {
					resultNote = "无法解析目标网站页面"
				} else if ts.AgentCode == _agent.AcTimeout {
					resultNote = "查询目标网站超时"
				} else {
					resultNote = "未知错误"
				}

				if resultStatus == 1 && ts.Err != "" {
					resultNote = resultNote + ": " + ts.Err
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
					if resultStatus == 0 {
						log.Printf("[INFO] Crawler %s(crawler-id=%d, carrier-code=%s, tracking-no=%s) is OK\n", crawlerInfo.Name, crawlerInfo.Id, crawlerInfo.CarrierCode, crawlerInfo.HeartBeatNo)
					} else {
						log.Printf("[WARN] Crawler %s(crawler-id=%d, carrier-code=%s, tracking-no=%s) has ERROR\n", crawlerInfo.Name, crawlerInfo.Id, crawlerInfo.CarrierCode, crawlerInfo.HeartBeatNo)
					}

					_db.SaveHealthLog(crawlerInfo.Id, ts.TrackingNo, int(timing), resultStatus, endTime, ts.AgentRawText, resultNote)
				}
			}
		}
	}
}
