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

func NewContactsInmem(cs ...*Contact) *ContactsInmem {
	index := make(map[ContactID]int, len(cs))
	for i, c := range cs {
		index[c.ID] = i
	}
	return &ContactsInmem{index: index, contacts: cs}
}

func (s *ContactsInmem) Create(_ context.Context, c *Contact) (ContactID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
retry:
	c.ID = ContactID{*new(uuid32).initV4()}
	_, loaded := s.index[c.ID]
	if loaded {
		goto retry
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
