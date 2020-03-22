package storage

import (
	"github.com/mmbednarek/smartapi/example"
)

type storage map[string]example.UserData

// NewStorage creates new storage
func NewStorage() example.Storage {
	return storage{}
}

// StoreUser stores a user
func (s storage) StoreUser(id string, data *example.UserData) error {
	if _, ok := s[id]; ok {
		return example.ErrUserAlreadyExists
	}
	s[id] = *data
	return nil
}

// GetUser gets a user
func (s storage) GetUser(id string) (*example.UserData, error) {
	user, ok := s[id]
	if !ok {
		return nil, example.ErrUserDoesNotExists
	}
	return &user, nil
}
