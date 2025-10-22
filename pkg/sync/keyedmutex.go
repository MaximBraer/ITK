package sync

import (
	"sync"
)

type KeyedMutex struct {
	mu     sync.Mutex
	mutexes map[string]*sync.Mutex
}

func NewKeyedMutex() *KeyedMutex {
	return &KeyedMutex{
		mutexes: make(map[string]*sync.Mutex),
	}
}

func (km *KeyedMutex) Lock(key string) {
	km.mu.Lock()
	mutex, exists := km.mutexes[key]
	if !exists {
		mutex = &sync.Mutex{}
		km.mutexes[key] = mutex
	}
	km.mu.Unlock()

	mutex.Lock()
}

func (km *KeyedMutex) Unlock(key string) {
	km.mu.Lock()
	mutex, exists := km.mutexes[key]
	km.mu.Unlock()

	if exists {
		mutex.Unlock()
	}
}

