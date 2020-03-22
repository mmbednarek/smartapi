package storage

import (
	"github.com/mmbednarek/smartapi/example"
)

type Storage map[string]example.UserData

func (s Storage) StoreUser(id string, data *example.UserData) error {
	if _, ok := s[id]; ok {
		return example.ErrUserAlreadyExists
	}
	s[id] = *data
	return nil
}

func (s Storage) GetUser(id string) (*example.UserData, error) {
	user, ok := s[id]
	if !ok {
		return nil, example.ErrUserDoesNotExists
	}
	return &user, nil
}
