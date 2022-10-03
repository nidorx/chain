package lib

import (
	"fmt"
	"sort"
	"strings"
)

var (
	ErrInvalidPattern      = fmt.Errorf("invalid pattern")
	ErrInvalidSplatPattern = fmt.Errorf("splat patterns must end with *")
	ErrItemAlreadyExist    = fmt.Errorf("item already exist")
)

// WildcardStore utility to persist and search items using wildcards. Used for channels, topics and events
//
// IMPORTANT: It is not safe for insertion in a concurrent scenario. Items should only persist during system startup.
type WildcardStore[T any] struct {
	exactly  map[string]T        // exactly match (Ex. /room:lobby)
	wildcard []*wildcardEntry[T] // wildcard match (Ex. /room:*)
}

type wildcardEntry[T any] struct {
	prefix string
	item   T
}

// Match returns the value corresponding to the first occurrence of the keyPattern that matches the given key
func (s *WildcardStore[T]) Match(key string) (out T) {
	if item, exist := s.exactly[key]; exist {
		out = item
		return
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
	if item, exist := s.exactly[key]; exist {
		items = append(items, item)
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

	keyPattern = strings.TrimSpace(keyPattern)

	if keyPattern == "" {
		return ErrInvalidPattern
	}

	// wildcard
	if strings.HasSuffix(keyPattern, "*") {
		prefix := strings.TrimSuffix(keyPattern, "*")

		if strings.ContainsRune(prefix, '*') {
			return ErrInvalidSplatPattern
		}

		for _, w := range s.wildcard {
			if w.prefix == prefix {
				return ErrItemAlreadyExist
			}
		}
		s.wildcard = append(s.wildcard, &wildcardEntry[T]{
			prefix: prefix,
			item:   value,
		})
		sort.Slice(s.wildcard, func(i, j int) bool {
			return len(s.wildcard[i].prefix) < len(s.wildcard[j].prefix)
		})
		return nil
	}

	if s.exactly == nil {
		s.exactly = map[string]T{}
	}

	if _, exist := s.exactly[keyPattern]; exist {
		return ErrItemAlreadyExist
	}

	s.exactly[keyPattern] = value
	return nil
}
