package pusher

import "encoding/json"

// DeviceRegister описывает информацию для регистрации нового токена устройства пользователя.
type DeviceRegister struct {
	App    string `json:"-"`      // идентификатор сервиса
	User   string `json:"user"`   // идентификатор пользователя
	Bundle string `json:"bundle"` // идентификатор приложения
	Token  string `json:"token"`  // идентификатор устройства
}

// PushMessage описывает информацию для отправки push-уведомления.
type PushMessage struct {
	App      string                     `json:"-"`        // идентификатор сервиса
	Users    []string                   `json:"users"`    // идентификаторы пользователей
	Messages map[string]json.RawMessage `json:"messages"` // сообщение для отправки с привязкой к идентификаторам приложения
}
