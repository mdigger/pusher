package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Notification описывает структуру push-уведомления.
type Notification struct {
	Token       string
	ID          string
	Expiration  time.Time
	LowPriority bool
	Topic       string
	CollapseID  string
	Payload     interface{}
	Sandbox     bool
}

// request возвращает сформированный запрос для отправки push-уведомления.
func (n *Notification) Request() (req *http.Request, err error) {
	var payload []byte
	switch data := n.Payload.(type) {
	case []byte:
		payload = data
	case string:
		payload = []byte(data)
	case json.RawMessage:
		payload = []byte(data)
	default:
		payload, err = json.Marshal(n.Payload)
		if err != nil {
			return nil, err
		}
	}
	var host = "https://api.push.apple.com"
	if n.Sandbox {
		host = "https://api.development.push.apple.com"
	}
	req, err = http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/3/device/%s", host, n.Token), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("user-agent", agent)
	req.Header.Set("content-type", "application/json")
	if n.ID != "" {
		req.Header.Set("apns-id", n.ID)
	}
	if !n.Expiration.IsZero() {
		var exp string = "0"
		if !n.Expiration.Before(time.Now()) {
			exp = strconv.FormatInt(n.Expiration.Unix(), 10)
		}
		req.Header.Set("apns-expiration", exp)
	}
	if n.LowPriority {
		req.Header.Set("apns-priority", "5")
	}
	if n.Topic != "" {
		req.Header.Set("apns-topic", n.Topic)
	}
	if n.CollapseID != "" && len(n.CollapseID) <= 64 {
		req.Header.Set("apns-collapse-id", n.CollapseID)
	}
	return req, nil
}
