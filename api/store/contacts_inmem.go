package store

import (
	"context"
	"sync"
)

type ContactsInmem struct {
	m sync.Map
}

var _ ContactsStore = new(ContactsInmem)

func (s *ContactsInmem) List(_ context.Context) ([]*Contact, error) {
	var contacts []*Contact
	s.m.Range(func(_, value any) bool {
		contacts = append(contacts, value.(*Contact))
		return true
	})
	return contacts, nil
}

func (s *ContactsInmem) Get(_ context.Context, id int) (*Contact, error) {
	value, ok := s.m.Load(id)
	if !ok {
		return nil, ErrObjectNotFound
	}
	return value.(*Contact), nil
}

func (s *ContactsInmem) Put(_ context.Context, id int, c *Contact) error {
	s.m.Store(id, c)
	return nil
}

func (s *ContactsInmem) Del(_ context.Context, id int) error {
	s.m.Delete(id)
	return nil
}
