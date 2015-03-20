package pusher

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type Handle func(string, http.ResponseWriter, *http.Request) (int, interface{})

// HTTPService описывает HTTP-сервис для работы отправки push-уведомлений и регистрации новых устройств.
type HTTPService struct {
	store  *Store  // хранилище
	config *Config // конфигурация
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
		store:  store,
		config: config,
	}
	for appId := range config.Apps {
		mux.HandleFunc(fmt.Sprintf("/%s", appId), handleWithData(appId, service.GetBundles))
		mux.HandleFunc(fmt.Sprintf("/%s/register", appId), handleWithData(appId, service.RegisterDevice))
		mux.HandleFunc(fmt.Sprintf("/%s/push", appId), handleWithData(appId, service.PushMessage))
	}
	mux.HandleFunc("/", service.GetApps)
	return service, nil
}

func handleWithData(appId string, handle Handle) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET": // TODO: Убрать после отладки
			fallthrough
		case "POST", "PUT":
			code, data := handle(appId, w, r)
			if code != http.StatusOK || code != http.StatusInternalServerError {
				code = http.StatusInternalServerError
			}
			jsonData, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				jsonData, err := json.MarshalIndent(err, "", "  ")
				if err != nil {
					http.Error(w, string(jsonData), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json; encoding=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(jsonData)
				return
			}
			w.Header().Set("Content-Type", "application/json; encoding=utf-8")
			w.WriteHeader(code)
			w.Write(jsonData)
		default:
			w.Header().Set("Allow", "POST,PUT")
			http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
			// http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

func (s *HTTPService) GetApps(w http.ResponseWriter, r *http.Request) {
	result := make([]string, 0, len(s.config.Apps))
	for app := range s.config.Apps {
		result = append(result, app)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; encoding=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *HTTPService) GetBundles(appId string, w http.ResponseWriter, r *http.Request) (int, interface{}) {
	result := make([]string, 0, len(s.config.Apps[appId]))
	for app := range s.config.Apps[appId] {
		result = append(result, app)
	}
	return http.StatusOK, result
}

func (s *HTTPService) RegisterDevice(appId string, w http.ResponseWriter, r *http.Request) (int, interface{}) {
	// Разбираем параметры запроса
	var regDevice *DeviceRegister
	switch mimetype := r.Header.Get("Content-Type"); {
	case strings.HasPrefix(mimetype, "application/json"): //json
		regDevice = new(DeviceRegister)
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(regDevice); err != nil {
			return http.StatusBadRequest, fmt.Sprintf("error parsing JSON request: %v", err)
		}
	default: // form
		regDevice = &DeviceRegister{ // form или get
			User:   r.FormValue("user"),
			Bundle: r.FormValue("bundle"),
			Token:  r.FormValue("token"),
		}
	}
	// сохраняем в хранилище
	if err := s.store.AddDevice(appId, regDevice.Bundle, regDevice.User, regDevice.Token); err != nil {
		return http.StatusInternalServerError, err.Error()
	}
	return http.StatusOK, regDevice
}

func (s *HTTPService) PushMessage(appId string, w http.ResponseWriter, r *http.Request) (int, interface{}) {
	userId := r.FormValue("user")
	devices, err := s.store.GetDevices(appId, userId)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}
	return http.StatusOK, devices
}

func (s *HTTPService) Close() error {
	return s.store.Close()
}
