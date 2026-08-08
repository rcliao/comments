package comment

import (
	"encoding/json"
	"os"
	"time"
)

// LatestReviewSince returns the most recent signoff recorded after t, or nil
// when no review has landed since. It reads the sidecar directly so polling
// for a review never triggers validation or re-anchoring side effects.
func LatestReviewSince(mdPath string, t time.Time) *ReviewRecord {
	data, err := os.ReadFile(GetSidecarPath(mdPath))
	if err != nil {
		return nil
	}
	var storage StorageFormat
	if err := json.Unmarshal(data, &storage); err != nil {
		return nil
	}
	if n := len(storage.Reviews); n > 0 {
		latest := storage.Reviews[n-1]
		if latest.Timestamp.After(t) {
			return &latest
		}
	}
	return nil
}
