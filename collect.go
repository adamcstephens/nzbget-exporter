package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	prom "github.com/prometheus/client_golang/prometheus"
)

type NZBGetCollector struct {
	Config *ExporterConfig

	version *prom.Desc

	articleCache    *prom.Desc
	downloadLimit   *prom.Desc
	downloadPaused  *prom.Desc
	downloadTimeSec *prom.Desc
	downloadedSize  *prom.Desc
	forcedSize      *prom.Desc
	freeDiskSpace   *prom.Desc
	postJobCount    *prom.Desc
	postPaused      *prom.Desc
	quotaDay        *prom.Desc
	quotaMonth      *prom.Desc
	quotaReached    *prom.Desc
	remainingSize   *prom.Desc
	resumeTime      *prom.Desc
	scanPaused      *prom.Desc
	serverStandBy   *prom.Desc
	startTime       *prom.Desc
	threadCount     *prom.Desc
	urlCount        *prom.Desc

	newsServerActive *prom.Desc
	newsServerBytes  *prom.Desc
}

func NewNZBGetCollector(config *ExporterConfig) *NZBGetCollector {
	ns := config.Namespace

	return &NZBGetCollector{
		Config: config,

		version: prom.NewDesc(
			prom.BuildFQName(ns, "", "version"),
			"always 1. label 'version' contains nzbget server version",
			[]string{"version"}, nil,
		),

		articleCache: prom.NewDesc(
			prom.BuildFQName(ns, "article_cache", "bytes"),
			"Current usage of article cache",
			nil, nil,
		),
		downloadLimit: prom.NewDesc(
			prom.BuildFQName(ns, "download", "limit"),
			"Current download limit, in bytes per second",
			nil, nil,
		),
		downloadPaused: prom.NewDesc(
			prom.BuildFQName(ns, "download", "paused"),
			"1 if the download queue is paused, 0 otherwise",
			nil, nil,
		),
		downloadTimeSec: prom.NewDesc(
			prom.BuildFQName(ns, "download", "time_seconds"),
			"Server download time in seconds",
			nil, nil,
		),
		downloadedSize: prom.NewDesc(
			prom.BuildFQName(ns, "downloaded", "total_bytes"),
			"Total data downloaded since server start",
			nil, nil,
		),
		forcedSize: prom.NewDesc(
			prom.BuildFQName(ns, "forced", "bytes"),
			"Remaining size of entries with FORCE priority",
			nil, nil,
		),
		freeDiskSpace: prom.NewDesc(
			prom.BuildFQName(ns, "disk", "free_bytes"),
			"Free disk space on 'DestDir'",
			nil, nil,
		),
		postJobCount: prom.NewDesc(
			prom.BuildFQName(ns, "post", "job_count"),
			"Number of Par-Jobs or Post-processing script jobs in the post-processing queue",
			nil, nil,
		),
		postPaused: prom.NewDesc(
			prom.BuildFQName(ns, "post", "active"),
			"1 if post-processor queue is currently active, 0 if paused",
			nil, nil,
		),
		quotaDay: prom.NewDesc(
			prom.BuildFQName(ns, "quota", "day_bytes"),
			"Daily quota in bytes", nil, nil,
		),
		quotaMonth: prom.NewDesc(
			prom.BuildFQName(ns, "quota", "month_bytes"),
			"Monthly quota in bytes", nil, nil,
		),
		quotaReached: prom.NewDesc(
			prom.BuildFQName(ns, "quota", "reached"),
			"1 if quota has been hit, 0 otherwise", nil, nil,
		),
		remainingSize: prom.NewDesc(
			prom.BuildFQName(ns, "queue", "remaining_bytes"),
			"Remaining size of all entries in download queue",
			nil, nil,
		),
		resumeTime: prom.NewDesc(
			prom.BuildFQName(ns, "resume", "time"),
			"Time to resume if set with method \"scheduleresume\"",
			nil, nil,
		),
		scanPaused: prom.NewDesc(
			prom.BuildFQName(ns, "scan", "active"),
			"1 if the scanning of incoming nzb-directory is currently active, 0 if paused",
			nil, nil,
		),
		serverStandBy: prom.NewDesc(
			prom.BuildFQName(ns, "", "standby"),
			"1 if no downloads in progress (server paused or all jobs completed), otherwise 0 if there are currently downloads running",
			nil, nil,
		),
		startTime: prom.NewDesc(
			prom.BuildFQName(ns, "start_time", "seconds"),
			"Server start time, in unixtime",
			nil, nil,
		),
		threadCount: prom.NewDesc(
			prom.BuildFQName(ns, "thread", "count"),
			"Number of threads running",
			nil, nil,
		),
		urlCount: prom.NewDesc(
			prom.BuildFQName(ns, "url", "count"),
			"Number of URLs in the URL-queue (including current file)",
			nil, nil,
		),

		newsServerActive: prom.NewDesc(
			prom.BuildFQName(ns, "news_server", "active"),
			"News server used for obtaining articles, 1 if active",
			[]string{"id", "server"}, nil,
		),
		newsServerBytes: prom.NewDesc(
			prom.BuildFQName(ns, "news_server", "total_bytes"),
			"Total bytes downloaded from this news server",
			[]string{"id", "server"}, nil,
		),
	}
}

func (c *NZBGetCollector) Collect(metrics chan<- prom.Metric) {
	var version string
	err := c.getApi("version", &version)
	if err != nil {
		log.WithError(err).Error("api get version")
		metrics <- prom.NewInvalidMetric(prom.NewInvalidDesc(err), err)
		return
	}
	metrics <- prom.MustNewConstMetric(c.version, prom.GaugeValue, 1, version)

	var s Status
	err = c.getApi("status", &s)
	if err != nil {
		log.WithError(err).Error("api get status")
		metrics <- prom.NewInvalidMetric(prom.NewInvalidDesc(err), err)
		return
	}
	metrics <- prom.MustNewConstMetric(c.articleCache, prom.GaugeValue, float64(s.ArticleCache))
	metrics <- prom.MustNewConstMetric(c.downloadLimit, prom.GaugeValue, float64(s.DownloadLimit))
	metrics <- prom.MustNewConstMetric(c.downloadPaused, prom.CounterValue, floatOf(s.DownloadPaused))
	metrics <- prom.MustNewConstMetric(c.downloadTimeSec, prom.GaugeValue, float64(s.DownloadTimeSec))
	metrics <- prom.MustNewConstMetric(c.downloadedSize, prom.CounterValue, float64(s.DownloadedSize))
	metrics <- prom.MustNewConstMetric(c.forcedSize, prom.GaugeValue, float64(s.ForcedSize))
	metrics <- prom.MustNewConstMetric(c.freeDiskSpace, prom.GaugeValue, float64(s.FreeDiskSpace))
	metrics <- prom.MustNewConstMetric(c.postJobCount, prom.GaugeValue, float64(s.PostJobCount))
	metrics <- prom.MustNewConstMetric(c.postPaused, prom.GaugeValue, floatOf(s.PostPaused))
	metrics <- prom.MustNewConstMetric(c.quotaDay, prom.GaugeValue, float64(s.DaySize))
	metrics <- prom.MustNewConstMetric(c.quotaMonth, prom.GaugeValue, float64(s.MonthSize))
	metrics <- prom.MustNewConstMetric(c.quotaReached, prom.GaugeValue, floatOf(s.QuotaReached))
	metrics <- prom.MustNewConstMetric(c.remainingSize, prom.GaugeValue, float64(s.RemainingSize))
	metrics <- prom.MustNewConstMetric(c.resumeTime, prom.GaugeValue, float64(s.ResumeTime.Unix()))
	metrics <- prom.MustNewConstMetric(c.scanPaused, prom.GaugeValue, floatOf(s.ScanPaused))
	metrics <- prom.MustNewConstMetric(c.serverStandBy, prom.GaugeValue, floatOf(s.ServerStandBy))
	metrics <- prom.MustNewConstMetric(c.startTime, prom.GaugeValue, float64(s.StartTime.Unix()))
	metrics <- prom.MustNewConstMetric(c.threadCount, prom.GaugeValue, float64(s.ThreadCount))
	metrics <- prom.MustNewConstMetric(c.urlCount, prom.GaugeValue, float64(s.URLCount))

	var config NZBGetConfig
	err = c.getApi("config", &config)
	if err != nil {
		log.WithError(err).Error("api get config")
		metrics <- prom.NewInvalidMetric(prom.NewInvalidDesc(err), err)
		return
	}

	var volume []ServerVolume
	err = c.getApi("servervolumes", &volume)
	if err != nil {
		log.WithError(err).Error("api get servervolumes")
		metrics <- prom.NewInvalidMetric(prom.NewInvalidDesc(err), err)
		return
	}

	// https://nzbget.net/api/servervolumes
	// NOTE: The first record (serverid=0) are totals for all servers
	for _, srv := range s.NewsServers {
		idx := srv.ID
		id := fmt.Sprintf("%d", srv.ID)
		name := config.Server[idx-1].Name
		active := floatOf(srv.Active)
		bytes := float64(volume[idx].TotalBytes)

		metrics <- prom.MustNewConstMetric(c.newsServerActive, prom.GaugeValue, active, id, name)
		metrics <- prom.MustNewConstMetric(c.newsServerBytes, prom.GaugeValue, bytes, id, name)
	}
}

func (c *NZBGetCollector) getApi(endpoint string, out interface{}) error {
	// Remove right-trailing slashes, otherwise NZBGet will 404
	host := strings.TrimRight(c.Config.Host, "/")

	u, err := url.Parse(host + "/jsonrpc/" + endpoint)
	if err != nil {
		return err
	}
	log.WithField("url", u.String()).Debug("GET api")
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}
	if c.Config.Username != "" && c.Config.Password != "" {
		req.SetBasicAuth(c.Config.Username, c.Config.Password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nzbget api response %d %s",
			resp.StatusCode, http.StatusText(resp.StatusCode),
		)
	}
	var response = new(Response)
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return err
	}

	return json.Unmarshal(response.Result, out)
}

func (c *NZBGetCollector) Describe(descr chan<- *prom.Desc) {
	descr <- c.articleCache
	descr <- c.downloadLimit
	descr <- c.downloadPaused
	descr <- c.downloadTimeSec
	descr <- c.downloadedSize
	descr <- c.forcedSize
	descr <- c.freeDiskSpace
	descr <- c.postJobCount
	descr <- c.postPaused
	descr <- c.quotaDay
	descr <- c.quotaMonth
	descr <- c.quotaReached
	descr <- c.remainingSize
	descr <- c.resumeTime
	descr <- c.scanPaused
	descr <- c.serverStandBy
	descr <- c.startTime
	descr <- c.threadCount
	descr <- c.urlCount

	descr <- c.newsServerActive
	descr <- c.newsServerBytes
}

var _ prom.Collector = &NZBGetCollector{}
