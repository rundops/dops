package web

import "context"

// contextWithCancel wraps context.WithCancel for the execution store.
func contextWithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}
