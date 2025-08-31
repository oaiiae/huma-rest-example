package datastores

import (
	"context"
	"slices"
	"sync"
)

// ContactsInmem implements [ContactsStore].
type ContactsInmem struct {
	mu       sync.Mutex
	index    map[ContactID]int
	contacts []*Contact
}

var _ ContactsStore = (*ContactsInmem)(nil)

func MustNewContactsInmem(cs ...*Contact) *ContactsInmem {
	s := &ContactsInmem{index: map[ContactID]int{}}
	for _, c := range cs {
		_, err := s.Create(context.Background(), c)
		if err != nil {
			panic(err)
		}
	}
	return s
}

func (s *ContactsInmem) Create(_ context.Context, c *Contact) (ContactID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ko := true; ko; _, ko = s.index[c.ID] {
		c.ID.initV4()
	}
	s.index[c.ID] = len(s.contacts)
	s.contacts = append(s.contacts, c)
	return c.ID, nil
}

func (s *ContactsInmem) List(_ context.Context, offset, length int) ([]*Contact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.contacts[min(offset, len(s.contacts)):min(offset+length, len(s.contacts))], nil
}

func (s *ContactsInmem) Get(_ context.Context, id ContactID) (*Contact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.index[id]
	if !ok || s.contacts[index] == nil {
		return nil, ErrObjectNotFound
	}
	return s.contacts[index], nil
}

func (s *ContactsInmem) Delete(_ context.Context, id ContactID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.index[id]
	if ok {
		delete(s.index, id)
		s.contacts = slices.Delete(s.contacts, index, index+1)
	}
	return nil
}
