package batch_mail

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"sync"
	"time"
)

type SimpleRateController struct {
	maxPerMinute     int           // max send per minute
	sentInLastMinute int           // sent in last minute
	lastResetTime    time.Time     // last reset time
	waitTime         time.Duration // wait time
	mu               sync.Mutex    // mutex

	// add field for more precise rate control
	lastSendTime time.Time // last send time
	sendTokens   int       // available send tokens; negative = reserved ahead by waiting workers
	burstLimit   int       // allowed burst send limit
	lastRefill   time.Time // when tokens were last topped up
	refillCarry  float64   // fractional token left over from the last top-up
}

// NewSimpleRateController creates a new SimpleRateController with the specified max send rate per minute.
func NewSimpleRateController(maxPerMinute int) *SimpleRateController {
	// parameter validation
	if maxPerMinute <= 0 {
		maxPerMinute = 1000
	}

	waitTime := time.Duration(float64(time.Minute) / float64(maxPerMinute) * 0.9) // reduce 10% wait time

	// ensure min wait time not too small, avoid system overload
	minWaitTime := 2 * time.Millisecond
	if waitTime < minWaitTime {
		waitTime = minWaitTime
	}

	// calculate burst limit, allow more send in short time
	burstLimit := maxPerMinute / 10
	if burstLimit < 10 {
		burstLimit = 10
	}

	g.Log().Debug(context.Background(), "create rate controller: %d per minute, wait time %v, burst limit %d",
		maxPerMinute, waitTime, burstLimit)

	return &SimpleRateController{
		maxPerMinute:  maxPerMinute,
		lastResetTime: time.Now(),
		waitTime:      waitTime,
		lastSendTime:  time.Now().Add(-waitTime), // ensure first send not need wait
		sendTokens:    burstLimit,                // initial tokens equal burst limit
		burstLimit:    burstLimit,
		lastRefill:    time.Now(),
	}
}

// Wait blocks until the next email may be sent.
//
// Token bucket: tokens refill at maxPerMinute, capped at burstLimit. The old version
// re-added "everything allowed so far this minute" on every call, so tokens refilled
// many times faster than the limit and campaigns went out all at once (#362).
// A worker that finds no token reserves the next one (tokens go negative) and sleeps
// until it is due, so concurrent workers queue up instead of all firing together.
func (r *SimpleRateController) Wait(ctx context.Context) error {
	r.mu.Lock()

	now := time.Now()
	perToken := time.Minute / time.Duration(r.maxPerMinute)

	// the per-minute counter only feeds GetCurrentRate/AdjustRate reporting
	if now.Sub(r.lastResetTime) >= time.Minute {
		r.sentInLastMinute = 0
		r.lastResetTime = now
	}

	if elapsed := now.Sub(r.lastRefill); elapsed > 0 {
		add := elapsed.Minutes()*float64(r.maxPerMinute) + r.refillCarry
		whole := int(add)
		r.refillCarry = add - float64(whole)
		r.sendTokens += whole
		if r.sendTokens >= r.burstLimit {
			r.sendTokens = r.burstLimit
			r.refillCarry = 0
		}
		r.lastRefill = now
	}

	r.sendTokens--
	if r.sendTokens >= 0 {
		r.lastSendTime = now
		r.mu.Unlock()
		return nil
	}

	// Our token is -sendTokens slots away, minus the fraction already accrued
	deficit := float64(-r.sendTokens) - r.refillCarry
	waitTime := time.Duration(deficit * float64(perToken))
	r.lastSendTime = now.Add(waitTime)
	r.mu.Unlock()

	if waitTime <= 0 {
		return nil
	}

	timer := time.NewTimer(waitTime)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		// give the reserved slot back
		r.mu.Lock()
		r.sendTokens++
		r.mu.Unlock()
		return ctx.Err()
	}
}

// RecordSend record one send
func (r *SimpleRateController) RecordSend() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// check if need reset counteed reset count
	now := time.Now()
	if now.Sub(r.lastResetTime) >= time.Minute {
		r.sentInLastMinute = 1
		r.lastResetTime = now
	} else {
		// add send count
		r.sentInLastMinute++
	}
}

// AdjustRate adjust send rate
func (r *SimpleRateController) AdjustRate() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsedSeconds := now.Sub(r.lastResetTime).Seconds()
	if elapsedSeconds < 5 {
		// collect at least 5 seconds data before adjust
		return
	}

	// projected rate in last minute
	projectedRate := float64(r.sentInLastMinute) / elapsedSeconds * 60
	targetRate := float64(r.maxPerMinute)

	if projectedRate > targetRate*1.2 {
		// send too fast, increase wait time
		r.waitTime = time.Duration(float64(r.waitTime) * 1)
		g.Log().Debug(context.Background(), "send rate too high (%.2f/minute), increase wait time to %v", projectedRate, r.waitTime)
	} else if projectedRate < targetRate*0.8 {
		// send too slow, reduce wait time (increase 20%)
		r.waitTime = time.Duration(float64(r.waitTime) * 0.8)
		g.Log().Debug(context.Background(), "send rate too low (%.2f/minute), reduce wait time to %v", projectedRate, r.waitTime)
	}

	// set min wait time, prevent too fast send
	minWait := time.Duration(2) * time.Millisecond
	if r.waitTime < minWait {
		r.waitTime = minWait
	}
}

// GetCurrentRate get current send rate
func (r *SimpleRateController) GetCurrentRate() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsedSeconds := now.Sub(r.lastResetTime).Seconds()
	if elapsedSeconds <= 0 {
		return 0
	}

	return float64(r.sentInLastMinute) / elapsedSeconds * 60
}

// GetMaxRate get max rate
func (r *SimpleRateController) GetMaxRate() int {
	return r.maxPerMinute
}
