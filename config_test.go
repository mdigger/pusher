package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mdigger/apns"
)

func TestConfig(t *testing.T) {
	var config Config
	config.SetAdmin("login", "password")
	for login, password := range map[string]string{
		"login":  "password",
		"login2": "password2",
		"login3": "password3",
		"login4": "password4",
		"login5": "password5",
	} {
		config.AddUser(login, password)
	}
	for certfile, password := range map[string]string{
		"cert.p12":         "xopen123",
		"cert2.p12":        "xopen123",
		"cert3.p12":        "open321",
		"cert4.p12":        "xopen123",
		"TiTPushDev.p12":   "xopen123",
		"TiTUniversal.p12": "xopen123",
	} {
		certificate, err := apns.LoadCertificate(certfile, password)
		if err != nil {
			t.Error("Load certificate error:", err)
			continue
		}
		info, replaced, err := config.Add(*certificate, false)
		if err != nil {
			t.Error("Add certificate error:", info)
			continue
		}
		if replaced {
			fmt.Println("Certificate replaced: ", certfile)
		}
	}
	config.filename = "certificate.gob"
	if err := config.Save(); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(config.filename)
	config2, err := LoadConfig(config.filename)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(config2.Users())
	if !config2.AuthorizeAdmin("login", "password") {
		t.Error("Failed admin authorization")
	}
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("Certificates:")
	for _, id := range config2.Certificates() {
		fmt.Println("-", id)
	}
	fmt.Println("Clients:")
	for _, id := range config2.Topics() {
		fmt.Println("-", id)
	}
	// client := config2.Client("com.xyzrd.trackintouch")
	// for _, token := range []string{
	// 	"BE311B5BADA725B323B1A56E03ED25B4814D6B9EDF5B02D3D605840860FEBB28", // iPad
	// 	"507C1666D7ECA6C26F40BC322A35CCB937E2BF02DFDACA8FCCAAD5CEE580EE8C", // iPad mini
	// 	"6B0420FA3B631DF5C13FB9DDC1BE8131C52B4E02580BB5F76BFA32862F284572", // iPhone
	// 	// "6B0420FA3B631DF5C13FB9DDC1BE8131C52B4E02580BB5F76BFA32862F284570", // Bad
	// } {
	// 	id, err := client.Push(apns.Notification{
	// 		Token:      token,
	// 		Expiration: time.Now().Add(-time.Hour),
	// 		Payload:    `{"aps":{"alert":"APNS test message"}}`,
	// 	})
	// 	fmt.Println(id)
	// 	if err != nil {
	// 		t.Error("Push error:", err)
	// 	}
	// }
}
