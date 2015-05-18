package models

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

var (
	// ErrMaxAttemptsExceeded indicates that the maximum retries has been
	// excedeed. Usually to consider a service unreachable/unavailable.
	ErrMaxAttemptsExceeded = errors.New("maximum of attempts exceeded")
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type Retry struct {
	// Attempts is the current number of attempts we did.
	Attempts int `bson:"attempts" json:"attempts"`
	// MaxAttempts is the maximum number of attempts we will try.
	MaxAttempts int `bson:"max_attempts" json:"maxAttempts"`
	// Factor is factor to increase the duration between each attempts.
	Factor float64 `bson:"factor" json:"factor"`
	// Min is the minimum duration between each attempts in seconds.
	Min int `bson:"min" json:"min"`
	// Max is the maximum duration between each attempts in seconds.
	Max int `bson:"max" json:"max"`
}

func (r *Retry) NextAttempt(now int64) (int64, error) {
	if r.MaxAttempts > 0 && r.Attempts+1 >= r.MaxAttempts {
		return 0, ErrMaxAttemptsExceeded
	}

	var next float64
	var min = float64(r.Min)
	var max = float64(r.Max)
	d := min * math.Pow(r.Factor, float64(r.Attempts))
	if d > max {
		next = max
	} else {
		next = d
	}
	// Randomize next run from 0% up to 20% of next interval.
	next += rand.Float64() * next / 5
	r.Attempts++
	return now + int64(next*1000000000), nil
}

func (r *Retry) SetDefault() {
	if r.MaxAttempts == 0 {
		r.MaxAttempts = 10
	}
	if r.Factor == 0 {
		r.Factor = 2
	}
	if r.Min == 0 {
		r.Min = 10
	}
	if r.Max == 0 {
		r.Max = 300
	}
}
