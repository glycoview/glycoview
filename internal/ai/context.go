package ai

import (
	"context"
	"time"
)

type tzCtxKey struct{}

// WithTimeZone stores the user's IANA timezone on the request context so tool
// handlers can anchor "today" / time-range computations to the user's local
// midnight rather than the server's UTC day.
func WithTimeZone(ctx context.Context, name string) context.Context {
	if name == "" {
		return ctx
	}
	return context.WithValue(ctx, tzCtxKey{}, name)
}

// LocationFromContext returns the configured timezone, or UTC if none is set
// / the name fails to load (e.g. Docker image without tzdata).
func LocationFromContext(ctx context.Context) *time.Location {
	name, _ := ctx.Value(tzCtxKey{}).(string)
	if name == "" {
		return time.UTC
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	return time.UTC
}
