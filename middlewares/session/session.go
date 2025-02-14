package session

type sessionState uint8

const (
	none sessionState = iota
	write
	drop
	renew
	ignore
)

type Session struct {
	state sessionState
	data  map[string]any
}

func (s *Session) Data() (data map[string]any) {
	data = make(map[string]any, len(s.data))
	for k, v := range s.data {
		data[k] = v
	}
	return
}

// Put puts the specified `value` in the session for the given `key`.
func (s *Session) Put(key string, value any) {
	if s.state == none {
		s.state = write
	}
	s.data[key] = value
}

// Get Returns session value for the given `key`. If `key` is not set, `nil` is returned.
func (s *Session) Get(key string) any {
	return s.data[key]
}

// Exist checks if the value exists in the data for that session
func (s *Session) Exist(key string) (exist bool) {
	_, exist = s.data[key]
	return
}

// GetMap Returns the whole session.
func (s *Session) GetMap() map[string]any {
	return s.data
}

// Delete Deletes `key` from session.
func (s *Session) Delete(key string) {
	if s.state == none {
		s.state = write
	}
	delete(s.data, key)
}

// Clear Clears the entire session.
//
// This function removes every key from the session, clearing the session.
//
// Note that, even if Clear is used, the session is still sent to the client. If the session should be
// effectively *dropped*, Destroy should be used.
func (s *Session) Clear() {
	if s.state == none {
		s.state = write
	}
	s.data = map[string]any{}
}

// Renew generates a new session id for the cookie
func (s *Session) Renew() {
	if s.state != ignore {
		s.state = renew
	}
}

// Destroy drops the session, a session cookie will not be included in the response
func (s *Session) Destroy() {
	if s.state != ignore {
		s.state = renew
	}
}

// IgnoreChanges ignores all changes made to the session in this request cycle
func (s *Session) IgnoreChanges() {
	s.state = ignore
}
