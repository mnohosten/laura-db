package metrics

import (
	"testing"
	"time"
)

func BenchmarkMetricsCollector_RecordQuery(b *testing.B) {
	mc := NewMetricsCollector()
	duration := 10 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.RecordQuery(duration, true)
	}
}

func BenchmarkMetricsCollector_RecordInsert(b *testing.B) {
	mc := NewMetricsCollector()
	duration := 5 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.RecordInsert(duration, true)
	}
}

func BenchmarkMetricsCollector_RecordUpdate(b *testing.B) {
	mc := NewMetricsCollector()
	duration := 7 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.RecordUpdate(duration, true)
	}
}

func BenchmarkMetricsCollector_RecordDelete(b *testing.B) {
	mc := NewMetricsCollector()
	duration := 3 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.RecordDelete(duration, true)
	}
}

func BenchmarkMetricsCollector_GetMetrics(b *testing.B) {
	mc := NewMetricsCollector()

	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		mc.RecordQuery(10*time.Millisecond, true)
		mc.RecordInsert(5*time.Millisecond, true)
		mc.RecordCacheHit()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mc.GetMetrics()
	}
}

func BenchmarkTimingHistogram_Record(b *testing.B) {
	th := NewTimingHistogram(1000)
	duration := 10 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		th.Record(duration)
	}
}

func BenchmarkTimingHistogram_GetBuckets(b *testing.B) {
	th := NewTimingHistogram(1000)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		th.Record(time.Duration(i) * time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = th.GetBuckets()
	}
}

func BenchmarkTimingHistogram_GetPercentiles(b *testing.B) {
	th := NewTimingHistogram(1000)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		th.Record(time.Duration(i) * time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = th.GetPercentiles()
	}
}

func BenchmarkMetricsCollector_Parallel(b *testing.B) {
	mc := NewMetricsCollector()
	duration := 10 * time.Millisecond

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mc.RecordQuery(duration, true)
		}
	})
}

func BenchmarkMetricsCollector_MixedOperations(b *testing.B) {
	mc := NewMetricsCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.RecordQuery(10*time.Millisecond, true)
		mc.RecordInsert(5*time.Millisecond, true)
		mc.RecordUpdate(7*time.Millisecond, true)
		mc.RecordDelete(3*time.Millisecond, true)
		mc.RecordCacheHit()
		mc.RecordIndexScan()
	}
}

func BenchmarkMetricsCollector_ConcurrentReads(b *testing.B) {
	mc := NewMetricsCollector()

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		mc.RecordQuery(10*time.Millisecond, true)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = mc.GetMetrics()
		}
	})
}

func BenchmarkMetricsCollector_ConcurrentWrites(b *testing.B) {
	mc := NewMetricsCollector()
	duration := 10 * time.Millisecond

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mc.RecordQuery(duration, true)
			mc.RecordInsert(duration, true)
			mc.RecordCacheHit()
		}
	})
}
