package metrics

import (
	"time"
)

// MetricsType 指标类型
type MetricsType string

const (
	// MetricsTypePrometheus Prometheus指标
	MetricsTypePrometheus MetricsType = "prometheus"
	// MetricsTypeInfluxDB InfluxDB指标
	MetricsTypeInfluxDB MetricsType = "influxdb"
	// MetricsTypeOpenTelemetry OpenTelemetry指标
	MetricsTypeOpenTelemetry MetricsType = "opentelemetry"
)

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	// IncCounter 增加计数器
	IncCounter(name string, labels map[string]string, value float64)
	// SetGauge 设置仪表盘
	SetGauge(name string, labels map[string]string, value float64)
	// ObserveHistogram 观察直方图
	ObserveHistogram(name string, labels map[string]string, value float64)
	// StartTimer 开始计时器
	StartTimer(name string, labels map[string]string) func()
	// Close 关闭指标收集器
	Close() error
}

// MetricsOptions 指标选项
type MetricsOptions struct {
	// Namespace 命名空间
	Namespace string
	// Subsystem 子系统
	Subsystem string
	// Address 地址
	Address string
	// Path 路径
	Path string
	// Labels 标签
	Labels map[string]string
}

// NewMetricsCollector 创建一个新的指标收集器
func NewMetricsCollector(metricsType MetricsType, options MetricsOptions) (MetricsCollector, error) {
	switch metricsType {
	case MetricsTypePrometheus:
		return NewPrometheusCollector(options)
	case MetricsTypeInfluxDB:
		return NewInfluxDBCollector(options)
	case MetricsTypeOpenTelemetry:
		return NewOpenTelemetryCollector(options)
	default:
		return NewPrometheusCollector(options)
	}
}

// CacheMetrics 缓存指标
type CacheMetrics struct {
	collector MetricsCollector
	namespace string
	subsystem string
	labels    map[string]string
}

// NewCacheMetrics 创建一个新的缓存指标
func NewCacheMetrics(collector MetricsCollector, namespace, subsystem string, labels map[string]string) *CacheMetrics {
	return &CacheMetrics{
		collector: collector,
		namespace: namespace,
		subsystem: subsystem,
		labels:    labels,
	}
}

// RecordHit 记录缓存命中
func (m *CacheMetrics) RecordHit() {
	m.collector.IncCounter("cache_hit", m.labels, 1)
}

// RecordMiss 记录缓存未命中
func (m *CacheMetrics) RecordMiss() {
	m.collector.IncCounter("cache_miss", m.labels, 1)
}

// RecordEviction 记录缓存淘汰
func (m *CacheMetrics) RecordEviction() {
	m.collector.IncCounter("cache_eviction", m.labels, 1)
}

// RecordExpiration 记录缓存过期
func (m *CacheMetrics) RecordExpiration() {
	m.collector.IncCounter("cache_expiration", m.labels, 1)
}

// RecordSize 记录缓存大小
func (m *CacheMetrics) RecordSize(size int64) {
	m.collector.SetGauge("cache_size", m.labels, float64(size))
}

// RecordItemCount 记录缓存项数量
func (m *CacheMetrics) RecordItemCount(count int) {
	m.collector.SetGauge("cache_item_count", m.labels, float64(count))
}

// RecordGetLatency 记录获取延迟
func (m *CacheMetrics) RecordGetLatency(d time.Duration) {
	m.collector.ObserveHistogram("cache_get_latency", m.labels, d.Seconds())
}

// RecordSetLatency 记录设置延迟
func (m *CacheMetrics) RecordSetLatency(d time.Duration) {
	m.collector.ObserveHistogram("cache_set_latency", m.labels, d.Seconds())
}

// RecordDeleteLatency 记录删除延迟
func (m *CacheMetrics) RecordDeleteLatency(d time.Duration) {
	m.collector.ObserveHistogram("cache_delete_latency", m.labels, d.Seconds())
}

// TimeGet 计时获取操作
func (m *CacheMetrics) TimeGet() func() {
	return m.collector.StartTimer("cache_get_latency", m.labels)
}

// TimeSet 计时设置操作
func (m *CacheMetrics) TimeSet() func() {
	return m.collector.StartTimer("cache_set_latency", m.labels)
}

// TimeDelete 计时删除操作
func (m *CacheMetrics) TimeDelete() func() {
	return m.collector.StartTimer("cache_delete_latency", m.labels)
}

// Close 关闭指标收集器
func (m *CacheMetrics) Close() error {
	return m.collector.Close()
}
