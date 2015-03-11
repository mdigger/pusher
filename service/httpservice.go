package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type HTTPService struct {
	store  *Store   // хранилище
	AppIds []string // идентификаторы сервисов приложений
}

func NewHTTPService(config *Config, mux *http.ServeMux) (*HTTPService, error) {
	if config == nil {
		return nil, errors.New("no config")
	}
	store, err := OpenStore(config.DB)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}
	service := &HTTPService{
		store: store,
	}
	appIds := make([]string, 0, len(config.Apps))
	for appId := range config.Apps {
		mux.HandleFunc(fmt.Sprintf("/%s/register", appId), handleWithData(appId, service.RegisterDevice))
		mux.HandleFunc(fmt.Sprintf("/%s/push", appId), handleWithData(appId, service.PushMessage))
		appIds = append(appIds, appId)
	}
	service.AppIds = appIds
	return service, nil
}

func handleWithData(appId string, handle func(string, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET": // TODO: Убрать после отладки
			fallthrough
		case "POST", "PUT":
			handle(appId, w, r)
		default:
			w.Header().Set("Allow", "POST,PUT")
			http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
			// http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

func (s *HTTPService) Close() error {
	return s.store.Close()
}

func (s *HTTPService) RegisterDevice(appId string, w http.ResponseWriter, r *http.Request) {
	// Разбираем параметры запроса
	var regDevice *DeviceRegister
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") { // json
		regDevice = new(DeviceRegister)
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(regDevice); err != nil {
			http.Error(w, fmt.Sprintf("error parsing JSON request: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		regDevice = &DeviceRegister{ // form или get
			UserId:     r.FormValue("userId"),
			DeviceType: r.FormValue("deviceType"),
			DeviceId:   r.FormValue("deviceId"),
		}
	}
	regDevice.AppId = appId                   // добавляем идентификатор сервиса
	if err := regDevice.Check(); err != nil { // проверяем правильность параметров
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.AddDevice(regDevice); err != nil { // сохраняем в хранилище
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("Registered:", regDevice)
	// w.WriteHeader(http.StatusNoContent)
	http.Error(w, http.StatusText(http.StatusOK), http.StatusOK) // возвращаем, что все хорошо
}

func (s *HTTPService) PushMessage(appId string, w http.ResponseWriter, r *http.Request) {
	userId := r.FormValue("userId")
	if userId == "" {
		http.Error(w, ErrDeviceRegistration_EmptyUserId.Error(), http.StatusBadRequest)
		return
	}
	devices, err := s.store.GetDevices(appId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := json.MarshalIndent(devices, "", "\t")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; encoding=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
