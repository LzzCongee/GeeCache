package metrics

import (
	"errors"
)

// InfluxDBCollector InfluxDB指标收集器
type InfluxDBCollector struct {
}

// NewInfluxDBCollector 创建一个新的InfluxDB指标收集器
func NewInfluxDBCollector(options MetricsOptions) (*InfluxDBCollector, error) {
	return nil, errors.New("InfluxDB metrics collector not implemented yet")
}

// IncCounter 增加计数器
func (c *InfluxDBCollector) IncCounter(name string, labels map[string]string, value float64) {
}

// SetGauge 设置仪表盘
func (c *InfluxDBCollector) SetGauge(name string, labels map[string]string, value float64) {
}

// ObserveHistogram 观察直方图
func (c *InfluxDBCollector) ObserveHistogram(name string, labels map[string]string, value float64) {
}

// StartTimer 开始计时器
func (c *InfluxDBCollector) StartTimer(name string, labels map[string]string) func() {
	return func() {}
}

// Close 关闭指标收集器
func (c *InfluxDBCollector) Close() error {
	return nil
}
