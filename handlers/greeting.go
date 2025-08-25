package handlers

import (
	"context"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
)

type Greeting struct{}

func (h *Greeting) RegisterAPI(api huma.API) { // called by [huma.AutoRegister]
	huma.Get(api, "/{name}", h.handle)
}

// GreetingOutput represents the greeting operation response.
type GreetingOutput struct {
	Body struct {
		Message string `json:"message" example:"Hello, world!" doc:"Greeting message"`
	}
}

func (h *Greeting) handle(_ context.Context, input *struct {
	Name string `path:"name" maxLength:"30" example:"world" doc:"Name to greet"`
}) (*GreetingOutput, error) {
	resp := &GreetingOutput{}
	resp.Body.Message = fmt.Sprintf("Hello, %s!", input.Name)
	return resp, nil
}
