package metrics

import (
	"errors"
)

// OpenTelemetryCollector OpenTelemetry指标收集器
type OpenTelemetryCollector struct {
}

// NewOpenTelemetryCollector 创建一个新的OpenTelemetry指标收集器
func NewOpenTelemetryCollector(options MetricsOptions) (*OpenTelemetryCollector, error) {
	return nil, errors.New("OpenTelemetry metrics collector not implemented yet")
}

// IncCounter 增加计数器
func (c *OpenTelemetryCollector) IncCounter(name string, labels map[string]string, value float64) {
}

// SetGauge 设置仪表盘
func (c *OpenTelemetryCollector) SetGauge(name string, labels map[string]string, value float64) {
}

// ObserveHistogram 观察直方图
func (c *OpenTelemetryCollector) ObserveHistogram(name string, labels map[string]string, value float64) {
}

// StartTimer 开始计时器
func (c *OpenTelemetryCollector) StartTimer(name string, labels map[string]string) func() {
	return func() {}
}

// Close 关闭指标收集器
func (c *OpenTelemetryCollector) Close() error {
	return nil
}
