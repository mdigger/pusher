package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMux(t *testing.T) {
	reqUrl := "/test"
	ts := httptest.NewServer(Handle("GET", reqUrl, func(c *Context) error {
		return c.Send("OK")
	}))
	resp, err := http.Get(ts.URL + reqUrl)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Error("bad handler")
	}
}
