package pusher

import (
	"testing"
)

func TestStore(t *testing.T) {
	store, err := OpenStore("test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	regDevice := &DeviceRegister{
		App:    "test",
		User:   "dmitrys",
		Bundle: "com.xyzrd.test",
		Token:  "token",
	}
	if err := store.AddDevice(regDevice); err != nil {
		t.Fatal(err)
	}
}
