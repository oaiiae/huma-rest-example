package store

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
	List(ctx context.Context) ([]*Contact, error)
	Get(ctx context.Context, id int) (*Contact, error)
	Put(ctx context.Context, id int, c *Contact) error
	Del(ctx context.Context, id int) error
}

var ErrObjectNotFound = errors.New("store: object not found")
