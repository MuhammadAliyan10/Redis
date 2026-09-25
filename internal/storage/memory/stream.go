package memory

import (
	"redis/internal/storage"
	"sync"
)




type Stream struct{
	mu sync.RWMutex
	messages []storage.StreamMessage
}

func NewStream() *Stream{
	return &Stream{
		messages: make([]storage.StreamMessage, 0),
	}
}


func (s *Stream) Append(value []byte) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := len(s.messages)
	s.messages = append(s.messages, storage.StreamMessage{
		ID : id,
		Value: value,
	})
	return  id
}

func (s *Stream) Read(id int) (storage.StreamMessage, bool){
	s.mu.RLock()
	defer s.mu.RUnlock()

	if id < 0 || id >= len(s.messages){
		return storage.StreamMessage{}, false
	}
	return s.messages[id], true

}
