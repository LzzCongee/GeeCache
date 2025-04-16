package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusCollector Prometheus指标收集器
type PrometheusCollector struct {
	registry   *prometheus.Registry
	namespace  string
	subsystem  string
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	server     *http.Server
	mu         sync.RWMutex
}

// NewPrometheusCollector 创建一个新的Prometheus指标收集器
func NewPrometheusCollector(options MetricsOptions) (*PrometheusCollector, error) {
	registry := prometheus.NewRegistry()
	collector := &PrometheusCollector{
		registry:   registry,
		namespace:  options.Namespace,
		subsystem:  options.Subsystem,
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}

	// 启动HTTP服务器
	if options.Address != "" {
		mux := http.NewServeMux()
		path := options.Path
		if path == "" {
			path = "/metrics"
		}
		mux.Handle(path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
		server := &http.Server{
			Addr:    options.Address,
			Handler: mux,
		}
		collector.server = server
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("Prometheus server error: %v\n", err)
			}
		}()
	}

	return collector, nil
}

// getCounter 获取计数器
func (c *PrometheusCollector) getCounter(name string) *prometheus.CounterVec {
	c.mu.RLock()
	counter, ok := c.counters[name]
	c.mu.RUnlock()
	if ok {
		return counter
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	counter, ok = c.counters[name]
	if ok {
		return counter
	}

	counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: c.namespace,
			Subsystem: c.subsystem,
			Name:      name,
			Help:      fmt.Sprintf("%s counter", name),
		},
		getLabelNames(nil),
	)
	c.registry.MustRegister(counter)
	c.counters[name] = counter
	return counter
}

// getGauge 获取仪表盘
func (c *PrometheusCollector) getGauge(name string) *prometheus.GaugeVec {
	c.mu.RLock()
	gauge, ok := c.gauges[name]
	c.mu.RUnlock()
	if ok {
		return gauge
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	gauge, ok = c.gauges[name]
	if ok {
		return gauge
	}

	gauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: c.namespace,
			Subsystem: c.subsystem,
			Name:      name,
			Help:      fmt.Sprintf("%s gauge", name),
		},
		getLabelNames(nil),
	)
	c.registry.MustRegister(gauge)
	c.gauges[name] = gauge
	return gauge
}

// getHistogram 获取直方图
func (c *PrometheusCollector) getHistogram(name string) *prometheus.HistogramVec {
	c.mu.RLock()
	histogram, ok := c.histograms[name]
	c.mu.RUnlock()
	if ok {
		return histogram
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	histogram, ok = c.histograms[name]
	if ok {
		return histogram
	}

	histogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: c.namespace,
			Subsystem: c.subsystem,
			Name:      name,
			Help:      fmt.Sprintf("%s histogram", name),
			Buckets:   prometheus.DefBuckets,
		},
		getLabelNames(nil),
	)
	c.registry.MustRegister(histogram)
	c.histograms[name] = histogram
	return histogram
}

// getLabelNames 获取标签名称
func getLabelNames(labels map[string]string) []string {
	if len(labels) == 0 {
		return []string{}
	}
	names := make([]string, 0, len(labels))
	for name := range labels {
		names = append(names, name)
	}
	return names
}

// getLabelValues 获取标签值
func getLabelValues(labels map[string]string) []string {
	if len(labels) == 0 {
		return []string{}
	}
	values := make([]string, 0, len(labels))
	for _, value := range labels {
		values = append(values, value)
	}
	return values
}

// IncCounter 增加计数器
func (c *PrometheusCollector) IncCounter(name string, labels map[string]string, value float64) {
	counter := c.getCounter(name)
	counter.With(labels).Add(value)
}

// SetGauge 设置仪表盘
func (c *PrometheusCollector) SetGauge(name string, labels map[string]string, value float64) {
	gauge := c.getGauge(name)
	gauge.With(labels).Set(value)
}

// ObserveHistogram 观察直方图
func (c *PrometheusCollector) ObserveHistogram(name string, labels map[string]string, value float64) {
	histogram := c.getHistogram(name)
	histogram.With(labels).Observe(value)
}

// StartTimer 开始计时器
func (c *PrometheusCollector) StartTimer(name string, labels map[string]string) func() {
	start := time.Now()
	return func() {
		c.ObserveHistogram(name, labels, time.Since(start).Seconds())
	}
}

// Close 关闭指标收集器
func (c *PrometheusCollector) Close() error {
	if c.server != nil {
		return c.server.Close()
	}
	return nil
}
