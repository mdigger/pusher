package service

import (
	"testing"
)

func TestConfig(t *testing.T) {
	config := &Config{
		DB:     "users.db",
		Server: "localhost:8080",
		Apps: map[string]*Application{
			"test-server":   &Application{},
			"test-server-2": &Application{},
			"test-server-3": &Application{},
		},
	}
	if err := config.Save("server/pusher.json"); err != nil {
		t.Error(err)
	}
}
