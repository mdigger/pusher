package service

import (
	"errors"
	"fmt"
)

// UniqueId хранит список уникальных идентификаторов.
type IdList []string

// Add добавляет новый уникальный идентификатор в список уникальных идентификаторов.
// Если такой идентификатор уже есть в списке, то возвращает false. В противном случае возвращает true.
func (ids *IdList) Add(deviceId string) bool {
	for _, id := range *ids {
		if id == deviceId {
			return false
		}
	}
	*ids = append(*ids, deviceId)
	return true
}

// Remove удаляет уникальный идентификатор из списка уникальных идентификаторов.
func (ids *IdList) Remove(deviceId string) {
	for i, id := range *ids {
		if id == deviceId {
			*ids = append((*ids)[:i], (*ids)[i+1:]...)
			return
		}
	}
}

// List возвращает список уникальных идентификаторов.
func (ids *IdList) List() []string {
	return []string(*ids)
}

// Devices описывает список токенов устройств пользователя.
type Devices struct {
	Apple  IdList `json:",omitempty"` // Apple iOS
	Google IdList `json:",omitempty"` // Google Android
}

// Add добавляет в список идентификатор устройства определенного типа.
// Возвращает true, если идентификатор добавлен.
func (d *Devices) Add(deviceType, deviceId string) bool {
	switch deviceType {
	case "apple":
		return d.Apple.Add(deviceId)
	case "google":
		return d.Google.Add(deviceId)
	}
	return false
}

func (d *Devices) Remove(deviceType, deviceId string) {
	switch deviceType {
	case "apple":
		d.Apple.Remove(deviceId)
	case "google":
		d.Google.Remove(deviceId)
	}
}

type DeviceRegister struct {
	AppId, // идентификатор сервиса
	UserId, // идентификатор пользователя
	DeviceType, // тип устройства (apple, google)
	DeviceId string // идентификатор устройства
}

func (dr *DeviceRegister) String() string {
	return fmt.Sprintf("[%s] %q — %q (%s)", dr.AppId, dr.UserId, dr.DeviceId, dr.DeviceType)
}

func (dr *DeviceRegister) Check() error {
	if dr.AppId == "" {
		return ErrDeviceRegistration_EmptyAppId
	}
	if dr.UserId == "" {
		return ErrDeviceRegistration_EmptyUserId
	}
	if dr.DeviceId == "" {
		return ErrDeviceRegistration_EmptyDeviceId
	}
	if dr.DeviceType != "apple" && dr.DeviceType != "google" {
		return ErrDeviceRegistration_BadDeviceType
	}
	return nil
}

var (
	ErrDeviceRegistration_EmptyAppId    = errors.New("empty application id")
	ErrDeviceRegistration_EmptyUserId   = errors.New("empty user id")
	ErrDeviceRegistration_EmptyDeviceId = errors.New("empty device id")
	ErrDeviceRegistration_BadDeviceType = errors.New("unknown device type")
)
