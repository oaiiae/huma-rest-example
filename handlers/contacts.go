package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	ds "github.com/oaiiae/huma-rest-example/datastores"
)

type Contacts struct {
	Store        ds.ContactsStore
	ErrorHandler func(context.Context, error)
}

type ContactModel struct {
	ID ds.ContactID `json:"id" readOnly:"true"`

	Firstname string `json:"firstname" example:"john"`
	Lastname  string `json:"lastname"  example:"smith"`
	Birthday  string `json:"birthday"  example:"1999-12-31" format:"date"`
}

func (h *Contacts) RegisterList(api huma.API) { // called by [huma.AutoRegister]
	huma.Get(api, "/",
		handlerWithErrorHandler(h.list, h.ErrorHandler),
		opErrors(http.StatusInternalServerError),
	)
}

type ContactsListOutput struct {
	Body []ContactModel
}

func (h *Contacts) list(ctx context.Context, _ *struct{}) (*ContactsListOutput, error) {
	contacts, err := h.Store.List(ctx)
	if err != nil {
		return nil, err
	}

	body := make([]ContactModel, 0, len(contacts))
	for _, contact := range contacts {
		body = append(body, ContactModel{
			ID:        contact.ID,
			Firstname: contact.Firstname,
			Lastname:  contact.Lastname,
			Birthday:  contact.Birthday.Format(time.DateOnly),
		})
	}

	return &ContactsListOutput{Body: body}, nil
}

func (h *Contacts) RegisterGet(api huma.API) { // called by [huma.AutoRegister]
	huma.Get(api, "/{id}",
		handlerWithErrorHandler(h.get, h.ErrorHandler),
		opErrors(http.StatusNotFound, http.StatusInternalServerError),
	)
}

type ContactsGetOutput struct {
	Body ContactModel
}

func (h *Contacts) get(ctx context.Context, input *struct {
	ID ds.ContactID `path:"id" doc:"ID of the contact to get"`
}) (*ContactsGetOutput, error) {
	contact, err := h.Store.Get(ctx, input.ID)
	switch {
	case err == nil:
		return &ContactsGetOutput{Body: ContactModel{
			ID:        contact.ID,
			Firstname: contact.Firstname,
			Lastname:  contact.Lastname,
			Birthday:  contact.Birthday.Format(time.DateOnly),
		}}, nil

	case errors.Is(err, ds.ErrObjectNotFound):
		return nil, huma.Error404NotFound("id not found", err)

	default:
		return nil, err
	}
}

func (h *Contacts) RegisterPut(api huma.API) { // called by [huma.AutoRegister]
	huma.Put(api, "/{id}",
		handlerWithErrorHandler(h.put, h.ErrorHandler),
		opErrors(http.StatusUnprocessableEntity, http.StatusInternalServerError),
	)
}

func (h *Contacts) put(ctx context.Context, input *struct {
	ID   ds.ContactID `path:"id" doc:"ID of the contact to put"`
	Body ContactModel
}) (*struct{}, error) {
	birthday, err := time.Parse(time.DateOnly, input.Body.Birthday)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("invalid format for birthday", err)
	}

	return nil, h.Store.Put(ctx, input.ID, &ds.Contact{
		ID:        input.ID,
		Firstname: input.Body.Firstname,
		Lastname:  input.Body.Lastname,
		Birthday:  birthday,
	})
}

func (h *Contacts) RegisterDel(api huma.API) { // called by [huma.AutoRegister]
	huma.Delete(api, "/{id}",
		handlerWithErrorHandler(h.del, h.ErrorHandler),
		opErrors(http.StatusInternalServerError),
	)
}

func (h *Contacts) del(ctx context.Context, input *struct {
	ID ds.ContactID `path:"id" doc:"ID of the contact to delete"`
}) (*struct{}, error) {
	return nil, h.Store.Del(ctx, input.ID)
}
