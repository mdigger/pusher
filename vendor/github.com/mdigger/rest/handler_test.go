package rest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerWithError(t *testing.T) {
	ts := httptest.NewServer(Handlers(Handler(func(c *Context) error {
		return errors.New("test error")
	})))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Error("bad handler with error")
	}
}
