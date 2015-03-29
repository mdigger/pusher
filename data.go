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

// Devices описывает список токенов устройств по идентификаторам приложений.
type Devices map[string][]string

// Add добавляет новый уникальный токен устройства в список уникальных идентификаторов.
// Если такой идентификатор уже есть в списке, то возвращает false. В противном случае возвращает
// true.
func (d Devices) Add(bundle, token string) bool {
	if _, ok := d[bundle]; !ok {
		d[bundle] = []string{token}
		return true
	}
	for _, id := range d[bundle] {
		if id == token {
			return false
		}
	}
	d[bundle] = append(d[bundle], token)
	return true
}

// Remove удаляет уникальный токен устройства из списка уникальных идентификаторов. В ответ
// возвращает true, если идентификатор удален, или false, если такого идентификатора небыло в
// списке.
func (d Devices) Remove(bundle, token string) bool {
	for i, id := range d[bundle] {
		if id == token {
			d[bundle] = append(d[bundle][:i], d[bundle][i+1:]...)
			return true
		}
	}
	return false
}

// List возвращает список уникальных токенов для указанного приложения.
func (d Devices) List(bundle string) []string {
	return d[bundle]
}
