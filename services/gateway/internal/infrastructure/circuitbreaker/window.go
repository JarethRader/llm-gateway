package circuitbreaker

import (
	"sync/atomic"
	"time"
)

type bucket struct {
	epoch    atomic.Int64 // which time-bucket index this slot currently represents
	success  atomic.Int64
	failures atomic.Int64
}

// window is a ring-buffer of N buckets covering *span*; each bucket covers span/N.
type window struct {
	buckets []bucket
	span    time.Duration
	width   time.Duration
}

func newWindow(span, width time.Duration, buckets int) *window {
	return &window{
		buckets: make([]bucket, buckets),
		span:    span,
		width:   width,
	}
}

func (w *window) record(now time.Time, ok bool) {
	idx, newEpoch := w.slot(now)
	bucket := &w.buckets[idx]
	oldEpoch := bucket.epoch.Load()
	if oldEpoch != newEpoch {
		for {
			current := bucket.epoch.Load()
			if current == newEpoch {
				break
			}
			if current == oldEpoch && bucket.epoch.CompareAndSwap(current, newEpoch) {
				bucket.success.Store(0)
				bucket.failures.Store(0)
				break
			}
			oldEpoch = current
		}
	}
	if ok {
		bucket.success.Add(1)
	} else {
		bucket.failures.Add(1)
	}
}

func (w *window) totals(now time.Time) (total, failures int) {
	cutoff := now.Add(-w.span)
	for i := range w.buckets {
		bucket := &w.buckets[i]
		if bucket.epoch.Load() == 0 {
			continue
		}
		if w.bucketTime(bucket.epoch.Load()).After(cutoff) {
			failures += int(bucket.failures.Load())
			total += int(bucket.success.Load()) + int(bucket.failures.Load())
		}
	}
	return
}

func (w *window) slot(now time.Time) (int64, int64) {
	epoch := now.UnixNano() / int64(w.width)
	idx := epoch % int64(len(w.buckets))
	return idx, epoch
}

func (w *window) bucketTime(epoch int64) time.Time {
	return time.Unix(0, epoch*int64(w.width))
}
