package pusher

type DeviceRegister struct {
	App    string `json:"-"`      // идентификатор сервиса
	User   string `json:"user"`   // идентификатор пользователя
	Bundle string `json:"bundle"` // идентификатор приложения
	Token  string `json:"token"`  // идентификатор устройства
}

// Devices описывает список токенов устройств по идентификаторам приложений.
type Devices map[string][]string

// Add добавляет новый уникальный идентификатор в список уникальных идентификаторов.
// Если такой идентификатор уже есть в списке, то возвращает false. В противном случае возвращает true.
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

// Remove удаляет уникальный идентификатор из списка уникальных идентификаторов.
func (d Devices) Remove(bundle, token string) bool {
	for i, id := range d[bundle] {
		if id == token {
			d[bundle] = append(d[bundle][:i], d[bundle][i+1:]...)
			return true
		}
	}
	return false
}

// List возвращает список уникальных идентификаторов.
func (d Devices) List(bundle string) []string {
	return d[bundle]
}
