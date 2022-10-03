package lib

import (
	"reflect"
	"testing"
)

var (
	sl1 = "blog"
	sl2 = "blog:category"
	sl3 = "blog:category:page"
	sl4 = "blog:category:page:subpage"
	sl5 = "blog:category:page:subpage:33"

	wl1 = "*"
	wl2 = "blog:*"
	wl3 = "blog:category:*"
	wl4 = "blog:category:page:*"
	wl5 = "blog:category:page:subpage:*"
)

func Test_WildcardStore_Match(t *testing.T) {

	routes := []struct {
		pattern string
		search  string
		result  any
	}{
		// exactly
		{sl1, sl1, true},
		{sl1, sl2, nil},
		{sl1, sl3, nil},
		{sl1, sl4, nil},
		{sl1, sl5, nil},

		{sl2, sl1, nil},
		{sl2, sl2, true},
		{sl2, sl3, nil},
		{sl2, sl4, nil},
		{sl2, sl5, nil},

		{sl3, sl1, nil},
		{sl3, sl2, nil},
		{sl3, sl3, true},
		{sl3, sl4, nil},
		{sl3, sl5, nil},

		{sl4, sl1, nil},
		{sl4, sl2, nil},
		{sl4, sl3, nil},
		{sl4, sl4, true},
		{sl4, sl5, nil},

		{sl5, sl1, nil},
		{sl5, sl2, nil},
		{sl5, sl3, nil},
		{sl5, sl4, nil},
		{sl5, sl5, true},

		// wildcards
		{wl1, sl1, true},
		{wl1, sl2, true},
		{wl1, sl3, true},
		{wl1, sl4, true},
		{wl1, sl5, true},

		{wl2, sl1, nil},
		{wl2, sl2, true},
		{wl2, sl3, true},
		{wl2, sl4, true},
		{wl2, sl5, true},

		{wl3, sl1, nil},
		{wl3, sl2, nil},
		{wl3, sl3, true},
		{wl3, sl4, true},
		{wl3, sl5, true},

		{wl4, sl1, nil},
		{wl4, sl2, nil},
		{wl4, sl3, nil},
		{wl4, sl4, true},
		{wl4, sl5, true},

		{wl5, sl1, nil},
		{wl5, sl2, nil},
		{wl5, sl3, nil},
		{wl5, sl4, nil},
		{wl5, sl5, true},
	}
	for _, tt := range routes {
		t.Run(tt.pattern, func(t *testing.T) {
			store := &WildcardStore[any]{}
			if err := store.Insert(tt.pattern, true); err != nil {
				t.Errorf("WildcardStore.Insert() | unexpected error %v", err)
			} else {
				value := store.Match(tt.search)
				if value != tt.result {
					t.Errorf("WildcardStore.Match() | invalid \n   actual: %v\n expected: %v", value, tt.result)
				}
			}
		})
	}
}

func Test_WildcardStore_MatchAll(t *testing.T) {

	store := &WildcardStore[string]{}
	store.Insert(sl1, "sl1")
	store.Insert(sl2, "sl2")
	store.Insert(sl3, "sl3")
	store.Insert(sl4, "sl4")
	store.Insert(sl5, "sl5")
	store.Insert(wl1, "wl1")
	store.Insert(wl2, "wl2")
	store.Insert(wl3, "wl3")
	store.Insert(wl4, "wl4")
	store.Insert(wl5, "wl5")

	tests := []struct {
		search string
		result []string
	}{
		{sl1, []string{"sl1", "wl1"}},
		{sl2, []string{"sl2", "wl1", "wl2"}},
		{sl3, []string{"sl3", "wl1", "wl2", "wl3"}},
		{sl4, []string{"sl4", "wl1", "wl2", "wl3", "wl4"}},
		{sl5, []string{"sl5", "wl1", "wl2", "wl3", "wl4", "wl5"}},
	}
	for _, tt := range tests {
		t.Run(tt.search, func(t *testing.T) {
			values := store.MatchAll(tt.search)
			if !reflect.DeepEqual(values, tt.result) {
				t.Errorf("WildcardStore.MatchAll() | invalid \n   actual: %v\n expected: %v", values, tt.result)
			}
		})
	}
}

func Test_WildcardStore_Errors(t *testing.T) {

	store := &WildcardStore[int]{}

	store.Insert("key", 0)
	store.Insert("wild*", 0)

	tests := []struct {
		key      string
		expected error
	}{
		{"   ", ErrInvalidPattern},
		{"ke*y*", ErrInvalidSplatPattern},
		{"key", ErrItemAlreadyExist},
		{"wild*", ErrItemAlreadyExist},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {

			err := store.Insert(tt.key, 0)
			if err != tt.expected {
				t.Errorf("WildcardStore.Insert() | invalid error \n   actual: %v\n expected: %v", err, tt.expected)
			}
		})
	}
}

func BenchmarkWildcardStore_Match(b *testing.B) {

	store := &WildcardStore[string]{}
	store.Insert(sl1, "sl1")
	store.Insert(sl2, "sl2")
	store.Insert(sl3, "sl3")
	store.Insert(sl4, "sl4")
	store.Insert(sl5, "sl5")
	store.Insert(wl1, "wl1")
	store.Insert(wl2, "wl2")
	store.Insert(wl3, "wl3")
	store.Insert(wl4, "wl4")
	store.Insert(wl5, "wl5")

	tests := []string{sl1, sl2, sl3, sl4, sl5}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, key := range tests {
			store.Match(key)
		}
	}
}

func BenchmarkWildcardStore_MatchAll(b *testing.B) {

	store := &WildcardStore[string]{}
	store.Insert(sl1, "sl1")
	store.Insert(sl2, "sl2")
	store.Insert(sl3, "sl3")
	store.Insert(sl4, "sl4")
	store.Insert(sl5, "sl5")
	store.Insert(wl1, "wl1")
	store.Insert(wl2, "wl2")
	store.Insert(wl3, "wl3")
	store.Insert(wl4, "wl4")
	store.Insert(wl5, "wl5")

	tests := []string{sl1, sl2, sl3, sl4, sl5}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, key := range tests {
			store.MatchAll(key)
		}
	}
}
