package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type Panic struct{}

func (h *Panic) RegisterAPI(api huma.API) { // called by [huma.AutoRegister]
	huma.Get(api, "/", h.handle)
}

func (h *Panic) handle(ctx context.Context, _ *struct{}) (*struct{}, error) {
	panic("panic argument")
}
