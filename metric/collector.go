package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type collector struct {
	registry *registry
	sampleCh chan ethrJSON
}

func newCollector(path string) *collector {
	c := new(collector)
	c.sampleCh = make(chan ethrJSON, 1)
	c.registry = newRegistry()
	go c.processSample()
	go c.process(path)
	return c
}

func (c collector) process(path string) {
	first := time.NewTicker(1 * time.Second)
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		// Run at the beginning (once)
		case <-first.C:
			if err := c.processFile(path); err != nil {
				log.Println(err)
			}
			first.Stop()
		// Runs every 1 min
		case <-ticker.C:
			if err := c.processFile(path); err != nil {
				log.Println(err)
			}
		}
	}
}

func (c collector) processFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var eJSON ethrJSON
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &eJSON); err != nil {
			return err
		}
		c.sampleCh <- eJSON
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (c collector) processSample() {
	for sample := range c.sampleCh {
		c.registry.mu.Lock()
		labels := prometheus.Labels{
			"AL":  sample.AverageLatency,
			"PPS": sample.PacketsPerSecond,
			"PT":  sample.Protocol,
			"BPS": sample.BitsPerSecond,
			"RA":  sample.RemoteAddr,
			"T":   sample.Type,
		}
		if r := c.registry.get("ethr_metric", labels); r != nil {
			c.registry.update(r, 0)
		} else {
			hash, labelNames := c.registry.hashLabels(labels)
			c.registry.storeMetric("ethr_metric", 0, prometheus.GaugeValue, hash, labels, labelNames)
		}
		c.registry.mu.Unlock()
	}
}

func (c collector) Collect(ch chan<- prometheus.Metric) {
	c.registry.mu.Lock()
	samples := make([]*registeredMetric, 0, c.registry.RefCount)
	for _, metric := range c.registry.Metrics {
		for _, rm := range metric {
			samples = append(samples, rm)
		}
	}
	c.registry.mu.Unlock()
	for _, sample := range samples {
		m, _ := prometheus.NewConstMetric(
			prometheus.NewDesc(sample.name, sample.help, sample.labelNames, prometheus.Labels{}),
			sample.valueType,
			sample.value,
			sample.labelValues...,
		)
		ch <- m
	}
}

func (c collector) Describe(_ chan<- *prometheus.Desc) {}
