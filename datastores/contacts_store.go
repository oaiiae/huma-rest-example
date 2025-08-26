package datastores

import (
	"context"
	"errors"
	"time"
)

type (
	ContactID struct{ uuid32 }
	Contact   struct {
		ID        ContactID
		Firstname string
		Lastname  string
		Birthday  time.Time
	}
)

type ContactsStore interface {
	Create(context.Context, *Contact) (ContactID, error)
	List(context.Context) ([]*Contact, error)
	Get(context.Context, ContactID) (*Contact, error)
	Delete(context.Context, ContactID) error
}

var ErrObjectNotFound = errors.New("store: object not found")
