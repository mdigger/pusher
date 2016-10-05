package main

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

func APNSError(status int, body io.Reader) error {
	var response = &Error{Status: status}
	if err := json.NewDecoder(body).Decode(response); err != nil {
		return err
	}
	return response
}

// Error describes the error response from the server.
type Error struct {
	Status    int
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}

// Error return full error description string.
func (e *Error) Error() string {
	msg, ok := reasons[e.Reason]
	if ok {
		return msg
	}
	msg = http.StatusText(e.Status)
	if msg == "" {
		msg = e.Reason
	}
	return msg
}

func (e *Error) Time() time.Time {
	if e.Timestamp == 0 {
		return time.Time{}
	}
	return time.Unix(e.Timestamp/1000, 0)
}

func (e *Error) IsToken() bool {
	switch e.Reason {
	case "BadDeviceToken",
		"MissingDeviceToken",
		"DeviceTokenNotForTopic",
		"TopicDisallowed",
		"Unregistered":
		return true
	default:
		return false
	}
}

var reasons = map[string]string{
	"BadCollapseId":               "The collapse identifier exceeds the maximum allowed size.",
	"BadDeviceToken":              "The specified device token was bad. Verify that the request contains a valid token and that the token matches the environment.",
	"BadExpirationDate":           "The apns-expiration value is bad.",
	"BadMessageId":                "The apns-id value is bad.",
	"BadPriority":                 "The apns-priority value is bad.",
	"BadTopic":                    "The apns-topic was invalid.",
	"DeviceTokenNotForTopic":      "The device token does not match the specified topic.",
	"DuplicateHeaders":            "One or more headers were repeated.",
	"IdleTimeout":                 "Idle time out.",
	"MissingDeviceToken":          "The device token is not specified in the request :path. Verify that the :path header contains the device token.",
	"MissingTopic":                "The apns-topic header of the request was not specified and was required. The apns-topic header is mandatory when the client is connected using a certificate that supports multiple topics.",
	"PayloadEmpty":                "The message payload was empty.",
	"TopicDisallowed":             "Pushing to this topic is not allowed.",
	"BadCertificate":              "The certificate was bad.",
	"BadCertificateEnvironment":   "The client certificate was for the wrong environment.",
	"ExpiredProviderToken":        "The provider token is stale and a new token should be generated.",
	"Forbidden":                   "The specified action is not allowed.",
	"InvalidProviderToken":        "The provider token is not valid or the token signature could not be verified.",
	"MissingProviderToken":        "No provider certificate was used to connect to APNs and Authorization header was missing or no provider token was specified.",
	"BadPath":                     "The request contained a bad :path value.",
	"MethodNotAllowed":            "The specified :method was not POST.",
	"Unregistered":                "The device token is inactive for the specified topic.",
	"PayloadTooLarge":             "The message payload was too large. See The Remote Notification Payload for details on maximum payload size.",
	"TooManyProviderTokenUpdates": "The provider token is being updated too often.",
	"TooManyRequests":             "Too many requests were made consecutively to the same device token.",
	"InternalServerError":         "An internal server error occurred.",
	"ServiceUnavailable":          "The service is unavailable.",
	"Shutdown":                    "The server is shutting down.",
}
