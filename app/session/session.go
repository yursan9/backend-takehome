package session

import (
	"math/rand/v2"
	"strings"
	"sync"
)

var (
	m            *sync.Mutex
	sessionStore map[string]struct{}
)

func Create() string {
	m.Lock()
	defer m.Unlock()

	id := generate(8)
	if _, ok := sessionStore[id]; !ok {
		sessionStore[id] = struct{}{}
	}
	return id
}

func Has(token string) bool {
	m.Lock()
	defer m.Unlock()

	_, ok := sessionStore[token]
	return ok
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
