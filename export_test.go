package paysafe

import (
	"context"
	"time"
)

// Test hooks kept out of the public API.
func WithTestSleep(f func(context.Context, time.Duration) error) Option { return withSleep(f) }
func WithTestRand(f func() float64) Option                              { return withRand(f) }
