package stats

import "time"

// This is a workaround to ensure we generate traffic at certain rate
// and stats are printed correctly. We ensure that current interval lasts
// 100ms after stats are printed, not perfect but workable.
func BeginThrottle(totalBytesToSend uint64, bufferLen int) (start time.Time, waitTime time.Duration, bytesToSend int) {
	start = time.Now()
	waitTime = time.Until(lastStatsTime.Add(time.Second + 50*time.Millisecond))
	bytesToSend = bufferLen
	if totalBytesToSend > 0 && totalBytesToSend < uint64(bufferLen) {
		bytesToSend = int(totalBytesToSend)
	}
	return
}

func EnforceThrottle(s time.Time, wt time.Duration, totalBytesToSend, oldSentBytes uint64, bufferLen int) (start time.Time, waitTime time.Duration, newSentBytes uint64, bytesToSend int) {
	start = s
	waitTime = wt
	newSentBytes = oldSentBytes
	bytesToSend = bufferLen
	if totalBytesToSend > 0 {
		remainingBytes := totalBytesToSend - oldSentBytes
		if remainingBytes > 0 {
			if remainingBytes < uint64(bufferLen) {
				bytesToSend = int(remainingBytes)
			}
		} else {
			timeTaken := time.Since(s)
			if timeTaken < wt {
				time.Sleep(wt - timeTaken)
			}
			start = time.Now()
			waitTime = time.Until(lastStatsTime.Add(time.Second + 50*time.Millisecond))
			newSentBytes = 0
			if totalBytesToSend < uint64(bufferLen) {
				bytesToSend = int(totalBytesToSend)
			}
		}
	}
	return
}

