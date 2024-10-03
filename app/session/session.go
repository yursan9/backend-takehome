package session

import (
	"math/rand/v2"
	"strings"
	"sync"
)

var (
	m            sync.Mutex
	sessionStore map[string]int = make(map[string]int)
)

func Create(id int) string {
	m.Lock()
	defer m.Unlock()

	key := generate(8)
	if _, ok := sessionStore[key]; !ok {
		sessionStore[key] = id
	}
	return key
}

func Get(token string) (int, bool) {
	m.Lock()
	defer m.Unlock()

	v, ok := sessionStore[token]
	return v, ok
}

func generate(n int) string {
	chars := "0123456789abcdefghijklmnopqrstuvwxyz"
	var sb strings.Builder
	sb.Grow(n)

	for i := 0; i < n; i++ {
		idx := rand.IntN(len(chars))
		sb.WriteByte(chars[idx])
	}
	return sb.String()
}
