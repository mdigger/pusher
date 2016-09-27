package rest

import (
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mdigger/router"
)

func NewContext(url string, compress bool) *Context {
	r, _ := http.NewRequest("GET", url, nil)
	if compress {
		r.Header.Set("Accept-Encoding", "gzip")
	}
	w := httptest.NewRecorder()
	return newContext(w, r)
}

func TestContext(t *testing.T) {
	Compress = true
	Debug = true
	c := NewContext("/test?test=test#sdf", true)
	defer c.close(nil)
	c.Request.Header.Set("Accept-Encoding", "gzip")

	header := c.Header()
	header.Set("test", "32")

	c.SetData("test", "value")
	if c.Data("test").(string) != "value" {
		t.Error("bad data set and get")
	}
	if err := c.Error(401, "test message"); err != nil {
		t.Error(err)
	}
	if c.Param("test") != "test" {
		t.Error("bad param")
	}
	c.WriteHeader(600)
	c.Write([]byte("test"))
	c.Flush()
	c.Send(nil)
}

func TestContext2(t *testing.T) {
	c := NewContext("/test", false)
	c.WriteHeader(600)
	c.close(nil)

	c = NewContext("/test", false)
	c.WriteHeader(0)
	c.close(nil)

	Compress = false
	c = NewContext("/test", false)
	c.Write([]byte("<html><h1>test</h1></html>"))
	c.Flush()
	c.close(nil)

	Compress = true
	c = NewContext("/test", true)
	c.Write([]byte("plain text"))
	c.Flush()
	c.close(nil)

	c = NewContext("/test", true)
	c.Send(nil)
	c.close(nil)

	c = NewContext("/test", true)
	c.Send([]byte("test"))
	c.close(nil)

	c = NewContext("/test", true)
	c.Send(errors.New("test"))
	c.close(nil)

	c = NewContext("/test", true)
	c.Send(strings.NewReader("test"))
	c.close(nil)

	c = NewContext("/test", true)
	c.Send("<html><h1>test</h1></html>")
	c.close(nil)

	c = NewContext("/test", true)
	c.Send("")
	c.close(nil)

	Compress = false
	c = NewContext("/test", true)
	c.Send([]byte("<html><h1>test</h1></html>"))
	c.Flush()
	c.close(nil)

	Compress = true
	c = NewContext("/test", true)
	c.Send("<html><h1>test</h1></html>")
	c.Flush()
	c.close(nil)

	c = NewContext("/test", true)
	c.Send(nil)
	c.Flush()
	c.close(nil)

	for _, err := range []error{
		errors.New("text"),
		ErrForbidden,
		ErrUnauthorized,
		ErrBadRequest,
		ErrLengthRequired,
		ErrNotFound,
		ErrNotImplemented,
		ErrServiceUnavailable,
		ErrDataAlreadySent,
		ErrInternalServerError,
		ErrUnsupportedMediaType,
		ErrRequestEntityTooLarge,
		io.EOF,
		io.ErrClosedPipe,
		io.ErrNoProgress,
		io.ErrShortWrite,
		io.ErrUnexpectedEOF,
		os.ErrExist,
		os.ErrInvalid,
		os.ErrPermission,
		os.ErrNotExist,
	} {
		c = NewContext("/test", true)
		Debug = (rand.Intn(10) < 5)
		EncodeError = (rand.Intn(10) >= 5)
		c.Send(err)
		c.close(nil)

		c = NewContext("/test", true)
		Debug = (rand.Intn(10) < 5)
		EncodeError = (rand.Intn(10) >= 5)
		c.Error(200+rand.Intn(301), err.Error())
		c.close(nil)
	}

	c = NewContext("/test", true)
	c.GetHeader("Context-Type")
	c.Redirect("/", http.StatusMovedPermanently)
	c.close(nil)

	c = NewContext("/test", true)
	c.SetCookie(&http.Cookie{Name: "test", Value: "test"})
	c.ServeFile("context_test.go")
	c.close(nil)

	c = NewContext("/test", true)
	c.ServeContent("test.txt", time.Now(), strings.NewReader("content"))
	c.Send(errors.New("text"))
	c.close(nil)

	// c = NewContext("/test", true)
	// c.Send("text")
	// go func() {
	// 	<-c.CloseNotify()
	// }()
	// c.close()

	// c = NewContext("/test", true)
	// c.Hijack()
	// c.Send("text")
	// c.close()

	c = NewContext("/test", true)
	c.close(nil)
}

func TestBind(t *testing.T) {
	jsontext := `{"test": "name"}`
	r, _ := http.NewRequest("POST", "/test", strings.NewReader(jsontext))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c := newContext(w, r)
	obj := make(map[string]string)
	err := c.Bind(&obj)
	c.close(err)
	if err != nil {
		t.Error(err)
	}

	r, _ = http.NewRequest("POST", "/test", strings.NewReader(jsontext))
	w = httptest.NewRecorder()
	c = newContext(w, r)
	obj = make(map[string]string)
	err = c.Bind(&obj)
	c.close(err)
	if err != ErrUnsupportedMediaType {
		t.Error("Error:", err)
	}

	r, _ = http.NewRequest("POST", "/test", strings.NewReader("{"+jsontext))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	c = newContext(w, r)
	obj = make(map[string]string)
	err = c.Bind(&obj)
	c.close(err)
	if err != ErrBadRequest {
		t.Error("Error:", err)
	}

	r, _ = http.NewRequest("POST", "/test", strings.NewReader(strings.Repeat("\"", 1<<16)))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	c = newContext(w, r)
	obj = make(map[string]string)
	err = c.Bind(&obj)
	c.close(err)
	if err != ErrRequestEntityTooLarge {
		t.Error("Error:", err)
	}
}

func TestParams(t *testing.T) {
	c := NewContext("/test", true)
	c.params = router.Params{
		{"name", "value"},
	}
	if c.Param("name") != "value" {
		t.Error("bad params")
	}
	c.Send(errors.New("text"))
	c.close(nil)
}
