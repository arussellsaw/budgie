package logging

import (
	"context"
	"sync"

	"github.com/monzo/slog"
)

type ContextParamLogger struct {
	slog.Logger
}

func (l ContextParamLogger) Log(evs ...slog.Event) {
	for i, e := range evs {
		params := Params(e.Context)
		if params == nil {
			continue
		}

		for k, v := range e.Metadata {
			params[k] = v
		}
		evs[i].Metadata = params
	}
	l.Logger.Log(evs...)
}

type paramKey string

type paramContainer struct {
	mu     sync.Mutex
	params map[string]string
}

func WithParams(ctx context.Context, params map[string]string) context.Context {
	return context.WithValue(ctx, paramKey("params"), paramContainer{params: params})
}

func SetParam(ctx context.Context, key, value string) context.Context {
	v, ok := ctx.Value(paramKey("params")).(paramContainer)
	if !ok {
		return WithParams(ctx, map[string]string{key: value})
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	v.params[key] = value

	return ctx
}

func Params(ctx context.Context) map[string]interface{} {
	container, ok := ctx.Value(paramKey("params")).(paramContainer)
	if !ok {
		return nil
	}

	container.mu.Lock()
	defer container.mu.Unlock()
	params := make(map[string]interface{})
	for k, v := range container.params {
		params[k] = v
	}

	return params
}
