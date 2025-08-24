package handler

import "context"

type handler[I, O any] = func(context.Context, *I) (*O, error)

func handlerWithErrorHandler[I, O any](handler handler[I, O], do func(context.Context, error)) handler[I, O] {
	if do == nil {
		return handler
	}

	return func(ctx context.Context, i *I) (*O, error) {
		o, err := handler(ctx, i)
		if err != nil {
			do(ctx, err)
		}
		return o, err
	}
}
