package snapshot

import (
	"context"
	"time"
)

const ctxCurrentTimeKey = "current_time_key"

func now(ctx context.Context) time.Time {
	return ctx.Value(ctxCurrentTimeKey).(time.Time)
}

func setNow(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxCurrentTimeKey, time.Now())
}

func mockNow(ctx context.Context, mockTime time.Time) context.Context {
	return context.WithValue(ctx, ctxCurrentTimeKey, mockTime)
}
