package app

import "context"

type contextKey string

const factoryKey contextKey = "factory"

func WithFactory(ctx context.Context, factory *Factory) context.Context {
	return context.WithValue(ctx, factoryKey, factory)
}

func FactoryFromContext(ctx context.Context) *Factory {
	if factory, ok := ctx.Value(factoryKey).(*Factory); ok {
		return factory
	}
	return nil
}
