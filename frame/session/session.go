package session

import "github.com/google/uuid"

type Session struct {
	Id    string
	data  map[string]interface{}
	IsNew bool
}

func (s *Session) SetAttribute(key string, value interface{}) {
	s.data[key] = value
}

func (s *Session) GetAttribute(key string) interface{} {
	return s.data[key]
}

func (s *Session) RemoveAttribute(key string) {
	delete(s.data, key)
}

func CreateNewSession() *Session {
	ss := &Session{
		Id:    uuid.New().String(),
		data:  make(map[string]interface{}),
		IsNew: true,
	}
	return ss
}
