package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mdigger/apns"
)

func TestAPNS(t *testing.T) {
	var APNSList APNS
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
		info, replaced, err := APNSList.Add(*certificate, false)
		if err != nil {
			t.Error("Add certificate error:", info)
			continue
		}
		if replaced {
			fmt.Println("Certificate replaced: ", certfile)
		}
	}
	if APNSList.SetSandbox("com.xyzrd.trackintouch.kid", true) {
		fmt.Println("Set Sandbox \"com.xyzrd.trackintouch.kid\"")
	}

	if APNSList.Remove("com.xyzrd.trackintouch.kid") {
		fmt.Println("Removed \"com.xyzrd.trackintouch.kid\"")
	}
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("Certificates:")
	for _, id := range APNSList.Certificates() {
		fmt.Println("-", id)
	}
	fmt.Println("Clients:")
	for _, id := range APNSList.Topics() {
		fmt.Println("-", id)
	}
	// client := APNSList.Client("com.xyzrd.trackintouch")
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
