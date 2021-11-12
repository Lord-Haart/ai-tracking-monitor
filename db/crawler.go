package db

import (
	"database/sql"
	"errors"
	"time"
)

type CrawlerInfoPo struct {
	CarrierId   int64  // 对应的运输商ID。
	CarrierCode string // 对应的运输商编号。

	HeartBeatNo string // 监控单号。
	Name        string // 查询代理名称。
	Url         string // 访问查询代理的URL。
	Type        string // 查询代理类型。

	Id                int64  // 查询代理ID。
	TargetUrl         string // 目标网页的URL。
	ReqHttpMethod     string // 访问目标网页的HTTP Method
	ReqHttpHeaders    string // 访问目标网页附带的头部。
	ReqHttpBody       string // 访问目标网页附带的数据。
	Verify            bool   // 是否需要验证请求结果。TODO: 以后删除。
	Json              bool   // 是否需要将payload序列化为json。TODO: 以后修改为 requestContentType
	ReqProxy          string // 代理服务器。
	ReqTimeout        int    // 访问目标网页的超时时间。
	SiteEncrypt       int    // 目标站点是否加密 0-不加密，1-需要加密。
	TrackingFieldName string // 附加字段名。
	TrackingFieldType int    // 附加字段类型。
	SiteCrawlingName  string
	SiteAnalyzedName  string
}

const (
	selectAllCrawlerInfo string = `	select ci.id, ci.carrier_code, tci.id, tci.heart_beat_no,
	tci.name, tci.req_url, tci.type, tcp.req_url, tcp.req_method, tcp.req_headers, tcp.req_data, tcp.req_verify, tcp.req_json, tcp.req_proxy,
	tcp.req_timeout, tcp.site_encrypt, tcp.tracking_field_name, tcp.tracking_field_type, tcp.site_crawling_name, tcp.site_analyzed_name
from tracking_crawler_info  tci
left join tracking_crawler_param tcp on tcp.info_id = tci.id
join carrier_info ci on ci.id = tci.carrier_id and ci.carrier_code is not null
where ci.status = 1
	and tci.status = 1
	and tci.heart_beat_no is not null
	and (tcp.status = 1 or tcp.status is null)
	and tci.service_status = 1
	and tci.start_time <= ?
	and tci.end_time >= ?
	and tci.type <> 'JAVA'
	order by tci.id, tci.priority`
)

func QueryAllCrawlerInfos(datePoint time.Time) []*CrawlerInfoPo {
	result := make([]*CrawlerInfoPo, 0)
	if rows, err := db.Query(selectAllCrawlerInfo, datePoint, datePoint); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result
		} else {
			panic(err)
		}
	} else {
		for rows.Next() {
			crawlerInfoPo := CrawlerInfoPo{}
			if rows.Scan(&crawlerInfoPo.CarrierId, &crawlerInfoPo.CarrierCode, &crawlerInfoPo.Id, &crawlerInfoPo.HeartBeatNo, &crawlerInfoPo.Name, &crawlerInfoPo.Url, &crawlerInfoPo.Type, &crawlerInfoPo.TargetUrl, &crawlerInfoPo.ReqHttpMethod, &crawlerInfoPo.ReqHttpHeaders, &crawlerInfoPo.ReqHttpBody,
				&crawlerInfoPo.Verify, &crawlerInfoPo.Json, &crawlerInfoPo.ReqProxy, &crawlerInfoPo.ReqTimeout, &crawlerInfoPo.SiteEncrypt, &crawlerInfoPo.TrackingFieldName, &crawlerInfoPo.TrackingFieldType, &crawlerInfoPo.SiteCrawlingName, &crawlerInfoPo.SiteAnalyzedName); err != nil {
				panic(err)
			} else {
				result = append(result, &crawlerInfoPo)
			}
		}

		return result
	}
}
