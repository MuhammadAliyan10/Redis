// redis/internal/storage/engine.go
package storage

import "time"

type StreamMessage struct{
	ID int
	Value []byte
}

type Value struct {
	Data []byte
	Type  string
	ExpiresAt time.Time
}

type Engine interface {
	Get(key string) (Value, bool)
	Set(key string, value []byte, ttl time.Duration)
	Del(key string) int
	Iterate(callback func(key string, value Value))

	XAdd(key string, value []byte) int
	XRead(key string, id int) (StreamMessage, bool)
}
