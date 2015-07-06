package pusher

import (
	"fmt"
	"strings"
	"testing"
)

func TestDB(t *testing.T) {
	store, err := OpenStore("test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.AddDevice("app", "bundle", "user1", "token"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddDevice("app", "bundle", "user2", "token"); err != nil {
		t.Fatal(err)
	}
	devices, err := store.GetDevices("app", "user1", "user2")
	if err != nil {
		t.Fatal(err)
	}
	for bundle, tokens := range devices {
		fmt.Println(bundle, strings.Join(tokens, ", "))
	}
}
