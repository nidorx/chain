package session

import (
	"github.com/syntax-framework/chain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func PerformRequest(router *chain.Router, method string, url string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	r, _ := http.NewRequest(method, url, nil)

	for _, cookie := range cookies {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

func Test_Store_Cookie(t *testing.T) {

	if err := chain.SetSecretKeyBase("ZcbD0D29eYsGq89QjirJbPkw7Qxwxboy"); err != nil {
		panic(err)
	}

	signature := ""
	router := chain.New()
	router.Use(&Manager{
		Config: Config{
			Key:  "sid",
			Path: "/",
		},
		Store: &Cookie{},
	})

	router.GET("/a", func(ctx *chain.Context) error {
		var sess *Session
		var err error
		if sess, err = FetchByKey(ctx, "sid"); err != nil {
			return err
		}

		sess.Put("value1", "X")
		sess.Put("value2", "Y")

		return nil
	})

	router.GET("/b", func(ctx *chain.Context) error {
		var sess *Session
		var err error
		if sess, err = FetchByKey(ctx, "sid"); err != nil {
			return err
		}

		value1 := sess.Get("value1")
		if value1 != nil {
			signature = signature + value1.(string)
		}

		value2 := sess.Get("value2")
		if value2 != nil {
			signature = signature + value2.(string)
		}
		return nil
	})

	w := PerformRequest(router, "GET", "/a", nil)
	if w.Code != http.StatusOK {
		t.Errorf("router.Use() failed: Invalid Code\n   actual: %v\n expected: %v", w.Code, http.StatusOK)
	}

	result := w.Result()
	cookies := result.Cookies()
	w = PerformRequest(router, "GET", "/b", cookies)
	if w.Code != http.StatusOK {
		t.Errorf("Store.Cookie failed: Invalid Code\n   actual: %v\n expected: %v", w.Code, http.StatusOK)
	}

	expected := "XY"
	if signature != expected {
		t.Errorf("Store.Cookie failed: Invalid Execution Order\n   actual: %v\n expected: %v", signature, expected)
	}
}
