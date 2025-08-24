package datastores

import (
	"context"
	"errors"
	"time"
)

type (
	ContactID = int
	Contact   struct {
		ID        ContactID
		Firstname string
		Lastname  string
		Birthday  time.Time
	}
)

type ContactsStore interface {
	List(context.Context) ([]*Contact, error)
	Get(context.Context, ContactID) (*Contact, error)
	Put(context.Context, ContactID, *Contact) error
	Del(context.Context, ContactID) error
}

var ErrObjectNotFound = errors.New("store: object not found")
