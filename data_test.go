package pusher

import (
	"github.com/kr/pretty"
	"testing"
)

func TestDevices(t *testing.T) {
	devices := make(Devices)
	devices.Add("test", "token")
	devices.Add("test", "token")
	devices.Add("test", "token2")
	devices.Add("test", "token3")
	devices.Remove("test", "token")
	pretty.Println("devices:", devices)
}
