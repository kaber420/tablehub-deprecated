package backoff

import (
	"context"
	"math/rand"
	"time"
)

// Config defines the configuration for the backoff retry strategy.
type Config struct {
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	MaxRetries int
	Multiplier float64
	Jitter     float64 // ratio from 0.0 to 1.0
}

// DefaultConfig returns a sane default backoff configuration.
func DefaultConfig() Config {
	return Config{
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   15 * time.Second,
		MaxRetries: 3,
		Multiplier: 2.0,
		Jitter:     0.1,
	}
}

// Retry executes the operation function with exponential backoff and jitter.
func Retry(ctx context.Context, cfg Config, operation func() error) error {
	var err error
	delay := cfg.BaseDelay

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if err = operation(); err == nil {
			return nil
		}

		if attempt == cfg.MaxRetries {
			break
		}

		// Calculate next delay with jitter
		actualDelay := float64(delay)
		if cfg.Jitter > 0 {
			// Apply a random variation between [-jitter * delay, +jitter * delay]
			jitterRange := cfg.Jitter * actualDelay
			minJitter := -jitterRange
			maxJitter := jitterRange
			randomJitter := minJitter + rand.Float64()*(maxJitter-minJitter)
			actualDelay += randomJitter
		}

		if actualDelay < 0 {
			actualDelay = float64(cfg.BaseDelay)
		}

		timer := time.NewTimer(time.Duration(actualDelay))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		// Increase delay for the next iteration
		nextDelay := float64(delay) * cfg.Multiplier
		if nextDelay > float64(cfg.MaxDelay) {
			delay = cfg.MaxDelay
		} else {
			delay = time.Duration(nextDelay)
		}
	}

	return err
}
