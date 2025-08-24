package datastores

import (
	"context"
	"sync"
)

// ContactsInmem implements [ContactsStore].
type ContactsInmem struct {
	m sync.Map
}

var _ ContactsStore = (*ContactsInmem)(nil)

func (s *ContactsInmem) With(cs ...Contact) *ContactsInmem {
	for _, c := range cs {
		s.m.Store(c.ID, &c)
	}
	return s
}

func (s *ContactsInmem) List(_ context.Context) ([]*Contact, error) {
	var contacts []*Contact
	s.m.Range(func(_, value any) bool {
		contacts = append(contacts, value.(*Contact))
		return true
	})
	return contacts, nil
}

func (s *ContactsInmem) Get(_ context.Context, id ContactID) (*Contact, error) {
	value, ok := s.m.Load(id)
	if !ok {
		return nil, ErrObjectNotFound
	}
	return value.(*Contact), nil
}

func (s *ContactsInmem) Put(_ context.Context, id ContactID, c *Contact) error {
	s.m.Store(id, c)
	return nil
}

func (s *ContactsInmem) Del(_ context.Context, id ContactID) error {
	s.m.Delete(id)
	return nil
}
