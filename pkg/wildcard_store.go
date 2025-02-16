package pkg

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

var (
	ErrInvalidPattern      = errors.New("invalid pattern")
	ErrItemAlreadyExist    = errors.New("item already exist")
	ErrInvalidSplatPattern = errors.New("splat patterns must end with *")
)

// WildcardStore utility to persist and search items using wildcards. Used for channels, topics and events
//
// IMPORTANT: Items should only persist during system startup.
type WildcardStore[T any] struct {
	mutex    sync.Mutex
	all      map[string]T        // all elements, by key
	exactly  map[string]T        // exactly match (Ex. /room:lobby)
	wildcard []*wildcardEntry[T] // wildcard match (Ex. /room:*)
}

type wildcardEntry[T any] struct {
	item   T
	prefix string
}

// Match returns the exactly value corresponding to the first occurrence of the keyPattern that matches the given key
func (s *WildcardStore[T]) Get(key string) (out T) {
	if s.all == nil {
		return
	}
	if item, exist := s.all[key]; exist {
		out = item
		return
	}

	return
}

// Match returns the value corresponding to the first occurrence of the keyPattern that matches the given key
func (s *WildcardStore[T]) Match(key string) (out T) {
	if s.exactly != nil {
		if item, exist := s.exactly[key]; exist {
			out = item
			return
		}
	}

	for _, entry := range s.wildcard {
		if len(entry.prefix) > len(key) {
			break
		}
		if entry.prefix == "" || strings.HasPrefix(key, entry.prefix) {
			out = entry.item
			return
		}
	}

	return
}

// MatchAll returns all existing values that match the given key
func (s *WildcardStore[T]) MatchAll(key string) []T {
	var items []T
	if s.exactly != nil {
		if item, exist := s.exactly[key]; exist {
			items = append(items, item)
		}
	}

	for _, entry := range s.wildcard {
		if len(entry.prefix) > len(key) {
			break
		}
		if entry.prefix == "" || strings.HasPrefix(key, entry.prefix) {
			items = append(items, entry.item)
		}
	}

	return items
}

func (s *WildcardStore[T]) Insert(keyPattern string, value T) error {

	if s.exactly == nil {
		s.exactly = map[string]T{}
	}

	if s.all == nil {
		s.all = map[string]T{}
	}

	if _, exist := s.all[keyPattern]; exist {
		return ErrItemAlreadyExist
	}

	keyPattern = strings.TrimSpace(keyPattern)

	if keyPattern == "" {
		return ErrInvalidPattern
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// wildcard
	if strings.ContainsRune(keyPattern, '*') {
		prefix := strings.TrimSuffix(keyPattern, "*")

		if strings.ContainsRune(prefix, '*') {
			return ErrInvalidSplatPattern
		}

		for _, w := range s.wildcard {
			if w.prefix == prefix {
				return ErrItemAlreadyExist
			}
		}

		wildcard := append(s.wildcard, &wildcardEntry[T]{prefix: prefix, item: value})
		sort.Slice(wildcard, func(i, j int) bool {
			return len(wildcard[i].prefix) < len(wildcard[j].prefix)
		})

		s.wildcard = wildcard
		s.all[keyPattern] = value
		return nil
	}

	s.all[keyPattern] = value
	s.exactly[keyPattern] = value

	return nil
}
