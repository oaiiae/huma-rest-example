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
	for i := range cs {
		_, err := s.Create(context.Background(), &cs[i])
		if err != nil {
			panic(err)
		}
	}
	return s
}

func (s *ContactsInmem) Create(_ context.Context, c *Contact) (ContactID, error) {
retry:
	c.ID = ContactID{*new(uuid32).initV4()}
	_, loaded := s.m.LoadOrStore(c.ID, c)
	if loaded {
		goto retry
	}
	return c.ID, nil
}

func (s *ContactsInmem) List(_ context.Context) ([]*Contact, error) {
	var contacts []*Contact
	s.m.Range(func(_, value any) bool {
		contacts = append(contacts, value.(*Contact)) //nolint: errcheck // always [*Contact]
		return true
	})
	return contacts, nil
}

func (s *ContactsInmem) Get(_ context.Context, id ContactID) (*Contact, error) {
	value, ok := s.m.Load(id)
	if !ok {
		return nil, ErrObjectNotFound
	}
	return value.(*Contact), nil //nolint: errcheck // always [*Contact]
}

func (s *ContactsInmem) Delete(_ context.Context, id ContactID) error {
	s.m.Delete(id)
	return nil
}
