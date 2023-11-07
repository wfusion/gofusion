package metrics

import (
	"math"
	"runtime"
	"strings"
	"time"

	"github.com/wfusion/gofusion/common/utils"

	iradix "github.com/hashicorp/go-immutable-radix"
)

type Label struct {
	Name  string
	Value string
}

func (m *Metrics) SetGauge(key []string, val float32, opts ...utils.OptionExtender) {
	m.SetGaugeWithLabels(key, val, nil, opts...)
}

func (m *Metrics) SetGaugeWithLabels(key []string, val float32, labels []Label, opts ...utils.OptionExtender) {
	key, labels, ok := m.formatKeyAndLabels("gauge", key, labels)
	if !ok {
		return
	}
	m.sink.SetGaugeWithLabels(key, val, labels, opts...)
}

func (m *Metrics) SetPrecisionGauge(key []string, val float64, opts ...utils.OptionExtender) {
	m.SetPrecisionGaugeWithLabels(key, val, nil, opts...)
}

func (m *Metrics) SetPrecisionGaugeWithLabels(key []string, val float64, labels []Label, opts ...utils.OptionExtender) {
	key, labels, ok := m.formatKeyAndLabels("gauge", key, labels)
	if !ok {
		return
	}
	sink, ok := m.sink.(PrecisionGaugeMetricSink)
	if !ok {
		m.sink.SetGaugeWithLabels(key, float32(val), labels, opts...)
	} else {
		sink.SetPrecisionGaugeWithLabels(key, val, labels, opts...)
	}
}

func (m *Metrics) EmitKey(key []string, val float32, opts ...utils.OptionExtender) {
	if m.EnableTypePrefix {
		key = insert(0, "kv", key)
	}
	if m.ServiceName != "" {
		key = insert(0, m.ServiceName, key)
	}
	_, allowed := m.allowMetric(key, nil)
	if !allowed {
		return
	}
	m.sink.EmitKey(key, val, opts...)
}

func (m *Metrics) IncrCounter(key []string, val float32, opts ...utils.OptionExtender) {
	m.IncrCounterWithLabels(key, val, nil, opts...)
}

func (m *Metrics) IncrCounterWithLabels(key []string, val float32, labels []Label, opts ...utils.OptionExtender) {
	key, labels, ok := m.formatKeyAndLabels("counter", key, labels)
	if !ok {
		return
	}
	m.sink.IncrCounterWithLabels(key, val, labels, opts...)
}

func (m *Metrics) AddSample(key []string, val float32, opts ...utils.OptionExtender) {
	m.AddSampleWithLabels(key, val, nil, opts...)
}

func (m *Metrics) AddSampleWithLabels(key []string, val float32, labels []Label, opts ...utils.OptionExtender) {
	key, labels, ok := m.formatKeyAndLabels("sample", key, labels)
	if !ok {
		return
	}
	m.sink.AddSampleWithLabels(key, val, labels, opts...)
}

func (m *Metrics) AddPrecisionSample(key []string, val float64, opts ...utils.OptionExtender) {
	m.AddPrecisionSampleWithLabels(key, val, nil, opts...)
}

func (m *Metrics) AddPrecisionSampleWithLabels(key []string, val float64,
	labels []Label, opts ...utils.OptionExtender) {
	key, labels, ok := m.formatKeyAndLabels("sample", key, labels)
	if !ok {
		return
	}
	if sink, ok := m.sink.(PrecisionSampleMetricSink); ok {
		sink.AddPrecisionSampleWithLabels(key, val, labels, opts...)
	} else {
		m.sink.AddSampleWithLabels(key, float32(val), labels, opts...)
	}
}

func (m *Metrics) MeasureSince(key []string, start time.Time, opts ...utils.OptionExtender) {
	m.MeasureSinceWithLabels(key, start, nil, opts...)
}

func (m *Metrics) MeasureSinceWithLabels(key []string, start time.Time, labels []Label, opts ...utils.OptionExtender) {
	key, labels, ok := m.formatKeyAndLabels("timer", key, labels)
	if !ok {
		return
	}
	opt := utils.ApplyOptions[Option](opts...)
	sink, ok := m.sink.(PrecisionSampleMetricSink)
	if m.TimerGranularity == 0 {
		if opt.Precision && ok {
			sink.AddPrecisionSampleWithLabels(key, math.MaxFloat64, labels, opts...)
		} else {
			m.sink.AddSampleWithLabels(key, math.MaxFloat32, labels, opts...)
		}
	} else {
		now := time.Now()
		elapsed := now.Sub(start)
		if opt.Precision && ok {
			msec := float64(elapsed.Nanoseconds()) / float64(m.TimerGranularity)
			sink.AddPrecisionSampleWithLabels(key, msec, labels, opts...)
		} else {
			msec := float32(elapsed.Nanoseconds()) / float32(m.TimerGranularity)
			m.sink.AddSampleWithLabels(key, msec, labels, opts...)
		}
	}
}

// UpdateFilter overwrites the existing filter with the given rules.
func (m *Metrics) UpdateFilter(allow, block []string) {
	m.UpdateFilterAndLabels(allow, block, m.AllowedLabels, m.BlockedLabels)
}

// UpdateFilterAndLabels overwrites the existing filter with the given rules.
func (m *Metrics) UpdateFilterAndLabels(allow, block, allowedLabels, blockedLabels []string) {
	m.filterLock.Lock()
	defer m.filterLock.Unlock()

	m.AllowedPrefixes = allow
	m.BlockedPrefixes = block

	if allowedLabels == nil {
		// Having a white list means we take only elements from it
		m.allowedLabels = nil
	} else {
		m.allowedLabels = make(map[string]bool)
		for _, v := range allowedLabels {
			m.allowedLabels[v] = true
		}
	}
	m.blockedLabels = make(map[string]bool)
	for _, v := range blockedLabels {
		m.blockedLabels[v] = true
	}
	m.AllowedLabels = allowedLabels
	m.BlockedLabels = blockedLabels

	m.filter = iradix.New()
	for _, prefix := range m.AllowedPrefixes {
		m.filter, _, _ = m.filter.Insert([]byte(prefix), true)
	}
	for _, prefix := range m.BlockedPrefixes {
		m.filter, _, _ = m.filter.Insert([]byte(prefix), false)
	}
}

func (m *Metrics) Shutdown() {
	if ss, ok := m.sink.(ShutdownSink); ok {
		ss.Shutdown()
	}
}

func (m *Metrics) formatKeyAndLabels(typePrefix string, keySrc []string, labelsSrc []Label) (
	keyDst []string, labelsDst []Label, ok bool) {
	keyDst = keySrc
	if m.HostName != "" {
		if m.EnableHostnameLabel {
			labelsSrc = append(labelsSrc, Label{"fus_hostname", m.HostName})
		} else if m.EnableHostname {
			keyDst = insert(0, m.HostName, keyDst)
		}
	}
	if m.EnableTypePrefix {
		keyDst = insert(0, typePrefix, keyDst)
	}
	if m.ServiceName != "" {
		if m.EnableServiceLabel {
			labelsSrc = append(labelsSrc, Label{"fus_service", m.ServiceName})
		} else {
			keyDst = insert(0, m.ServiceName, keyDst)
		}
	}
	if m.EnableClientIPLabel {
		labelsSrc = append(labelsSrc, Label{"fus_service_ip", utils.ClientIP()})
	}

	labelsDst, ok = m.allowMetric(keyDst, labelsSrc)
	return
}

// labelIsAllowed return true if a should be included in metric
// the caller should lock m.filterLock while calling this method
func (m *Metrics) labelIsAllowed(label *Label) bool {
	labelName := (*label).Name
	if m.blockedLabels != nil {
		_, ok := m.blockedLabels[labelName]
		if ok {
			// If present, let's remove this label
			return false
		}
	}
	if m.allowedLabels != nil {
		_, ok := m.allowedLabels[labelName]
		return ok
	}
	// Allow by default
	return true
}

// filterLabels return only allowed labels
// the caller should lock m.filterLock while calling this method
func (m *Metrics) filterLabels(labels []Label) []Label {
	if labels == nil {
		return nil
	}
	toReturn := make([]Label, 0, len(labels))
	for _, label := range labels {
		if m.labelIsAllowed(&label) {
			toReturn = append(toReturn, label)
		}
	}
	return toReturn
}

// Returns whether the metric should be allowed based on configured prefix filters
// Also return the applicable labels
func (m *Metrics) allowMetric(key []string, labels []Label) ([]Label, bool) {
	m.filterLock.RLock()
	defer m.filterLock.RUnlock()

	if m.filter == nil || m.filter.Len() == 0 {
		return m.filterLabels(labels), m.Config.FilterDefault
	}

	_, allowed, ok := m.filter.Root().LongestPrefix([]byte(strings.Join(key, ".")))
	if !ok {
		return m.filterLabels(labels), m.Config.FilterDefault
	}

	return m.filterLabels(labels), allowed.(bool)
}

// Periodically collects runtime stats to publish
func (m *Metrics) collectStats(opts ...utils.OptionExtender) {
	for {
		time.Sleep(m.ProfileInterval)
		m.EmitRuntimeStats(opts...)
	}
}

// EmitRuntimeStats various runtime statsitics
func (m *Metrics) EmitRuntimeStats(opts ...utils.OptionExtender) {
	// avoid panic
	_, _ = utils.Catch(func() {
		// Export number of Goroutines
		numRoutines := runtime.NumGoroutine()
		m.SetGauge([]string{"runtime", "num_goroutines"}, float32(numRoutines))

		// Export memory stats
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		m.SetGauge([]string{"runtime", "alloc_bytes"}, float32(stats.Alloc), opts...)
		m.SetGauge([]string{"runtime", "sys_bytes"}, float32(stats.Sys), opts...)
		m.SetGauge([]string{"runtime", "malloc_count"}, float32(stats.Mallocs), opts...)
		m.SetGauge([]string{"runtime", "free_count"}, float32(stats.Frees), opts...)
		m.SetGauge([]string{"runtime", "heap_objects"}, float32(stats.HeapObjects), opts...)
		m.SetGauge([]string{"runtime", "total_gc_pause_ns"}, float32(stats.PauseTotalNs), opts...)
		m.SetGauge([]string{"runtime", "total_gc_runs"}, float32(stats.NumGC), opts...)

		// Export info about the last few GC runs
		num := stats.NumGC

		// Handle wrap around
		if num < m.lastNumGC {
			m.lastNumGC = 0
		}

		// Ensure we don't scan more than 256
		if num-m.lastNumGC >= 256 {
			m.lastNumGC = num - 255
		}

		for i := m.lastNumGC; i < num; i++ {
			pause := stats.PauseNs[i%256]
			m.AddSample([]string{"runtime", "gc_pause_ns"}, float32(pause))
		}
		m.lastNumGC = num
	})
}

// Creates a new slice with the provided string value as the first element
// and the provided slice values as the remaining values.
// Ordering of the values in the provided input slice is kept in tact in the output slice.
func insert(i int, v string, s []string) []string {
	// Allocate new slice to avoid modifying the input slice
	newS := make([]string, len(s)+1)

	// Copy s[0, i-1] into newS
	for j := 0; j < i; j++ {
		newS[j] = s[j]
	}

	// Insert provided element at index i
	newS[i] = v

	// Copy s[i, len(s)-1] into newS starting at newS[i+1]
	for j := i; j < len(s); j++ {
		newS[j+1] = s[j]
	}

	return newS
}
