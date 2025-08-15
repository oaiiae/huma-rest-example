package handler

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/oaiiae/huma-rest-example/store"
)

type Contacts struct {
	Store store.ContactsStore
}

func (h *Contacts) RegisterAPI(api huma.API) { // called by [huma.AutoRegister]
	api = huma.NewGroup(api, "/contacts")
	huma.Get(api, "/", h.list)
	huma.Get(api, "/{id}", h.get)
	huma.Put(api, "/{id}", h.put)
	huma.Delete(api, "/{id}", h.del)
}

type ContactModel struct {
	ID        int    `json:"id" example:"12" readOnly:"true"`
	Firstname string `json:"firstname" example:"john"`
	Lastname  string `json:"lastname" example:"smith"`
	Birthday  string `json:"birthday" format:"date" example:"1999-12-31"`
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

type ContactsGetOutput struct {
	Body ContactModel
}

func (h *Contacts) get(ctx context.Context, input *struct {
	ID int `path:"id" example:"12" doc:"ID of the contact to get"`
}) (*ContactsGetOutput, error) {
	contact, err := h.Store.Get(ctx, input.ID)
	switch err {
	case store.ErrObjectNotFound:
		return nil, huma.Error404NotFound("", err)
	case nil:
		return &ContactsGetOutput{Body: ContactModel{
			ID:        contact.ID,
			Firstname: contact.Firstname,
			Lastname:  contact.Lastname,
			Birthday:  contact.Birthday.Format(time.DateOnly),
		}}, nil
	default:
		return nil, err
	}
}

func (h *Contacts) put(ctx context.Context, input *struct {
	ID   int `path:"id" example:"12" doc:"ID of the contact to put"`
	Body ContactModel
}) (*struct{}, error) {
	birthday, err := time.Parse(time.DateOnly, input.Body.Birthday)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid birthday", err)
	}

	return nil, h.Store.Put(ctx, input.ID, &store.Contact{
		ID:        input.ID,
		Firstname: input.Body.Firstname,
		Lastname:  input.Body.Lastname,
		Birthday:  birthday,
	})
}

func (h *Contacts) del(ctx context.Context, input *struct {
	ID int `path:"id" example:"12" doc:"ID of the contact to delete"`
}) (*struct{}, error) {
	return nil, h.Store.Del(ctx, input.ID)
}
