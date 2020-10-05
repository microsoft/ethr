package main

import (
	"bytes"
	"hash"
	"hash/fnv"
	"sort"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

type (
	valueHash uint64
	Metrics   map[valueHash]*registeredMetric
)

type registeredMetric struct {
	name      string
	value     float64
	help      string
	valueType prometheus.ValueType

	// Keep LabelNames and LabelValues separately as it'll be
	// required by prometheus.MustNewConstMetric in the Collect
	// method
	labelNames  []string
	labelValues []string

	// Hash of label name + label value pairs
	valueKey valueHash
}

type registry struct {
	// mu holds lock on Metrics
	mu sync.RWMutex
	// RefCount indicates total number of registered metrics currently present
	// in store
	RefCount int64
	// Metrics holds all combination of label name-value pairs in map that corresponds
	// to a metric name (key).
	// metricHolder holds the type of metric and the map of registered metrics
	Metrics map[string]Metrics
	// The below value and label variables are allocated in the registry struct
	// so that we don't have to allocate them every time have to compute a label
	// hash.
	valueBuf, nameBuf bytes.Buffer
	// hasher function
	hasher hash.Hash64
}

func (r *registry) hashLabels(labels prometheus.Labels) (valueHash, []string) {
	r.hasher.Reset()
	r.nameBuf.Reset()
	r.valueBuf.Reset()
	labelNames := make([]string, 0, len(labels))
	for labelName := range labels {
		labelNames = append(labelNames, labelName)
	}
	sort.Strings(labelNames)
	r.valueBuf.WriteByte(model.SeparatorByte)
	for _, labelName := range labelNames {
		// Label Value
		r.valueBuf.WriteString(labels[labelName])
		r.valueBuf.WriteByte(model.SeparatorByte)
		// Label Key
		r.nameBuf.WriteString(labelName)
		r.nameBuf.WriteByte(model.SeparatorByte)
	}
	r.hasher.Write(r.nameBuf.Bytes())
	r.hasher.Write(r.valueBuf.Bytes())
	return valueHash(r.hasher.Sum64()), labelNames
}

func (r *registry) getMetric(metricName string) *Metrics {
	metric, hasMetric := r.Metrics[metricName]
	if hasMetric {
		return &metric
	}
	return nil
}

func (m Metrics) getRM(hash valueHash) *registeredMetric {
	rm, ok := m[hash]
	if ok {
		return rm
	}
	return nil
}

func (r *registry) get(metricName string, labels prometheus.Labels) *registeredMetric {
	if m := r.getMetric(metricName); m != nil {
		hash, _ := r.hashLabels(labels)
		if r := m.getRM(hash); r != nil {
			return r
		}
	}
	return nil
}

func (r *registry) storeMetric(metricName string, value float64, metricType prometheus.ValueType, hash valueHash, labels prometheus.Labels, labelNames []string) {
	if r.getMetric(metricName) == nil {
		r.Metrics[metricName] = make(map[valueHash]*registeredMetric)
	}
	r.storeRM(r.Metrics[metricName], metricName, value, metricType, hash, labels, labelNames)
}

func (r *registry) storeRM(m Metrics, metricName string, value float64, metricType prometheus.ValueType, hash valueHash, labels prometheus.Labels, labelNames []string) {
	if m.getRM(hash) == nil {
		r.RefCount++
		labelValues := make([]string, 0, len(labelNames))
		for _, labelName := range labelNames {
			labelValues = append(labelValues, labels[labelName])
		}
		rm := &registeredMetric{
			valueKey:    hash,
			name:        metricName,
			value:       value,
			valueType:   metricType,
			labelNames:  labelNames,
			labelValues: labelValues,
		}
		// Add entry to store
		m[hash] = rm
	}
}

// update is used to update an existing registered metric by updating it's value
func (r *registry) update(rm *registeredMetric, value float64) {
	rm.value = value
}

func newRegistry() *registry {
	return &registry{
		Metrics: make(map[string]Metrics),
		hasher:  fnv.New64a(),
	}
}
