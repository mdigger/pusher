package pusher

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/alexjlockwood/gcm"
	"github.com/mdigger/apns"
)

const tokenName = "Token "

// Handle описывает формат обработчика HTTP-запросов, поддерживаемый сервером.
type Handle func(string, http.ResponseWriter, *http.Request) (int, interface{})

// HTTPService описывает HTTP-сервис для работы отправки push-уведомлений и регистрации новых
// устройств.
type HTTPService struct {
	store  *Store  // хранилище
	config *Config // конфигурация
}

// NewHTTPService инициализирует обработчики HTTP-запросов для всех определенных в конфигурации
// сервисов.
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
	mux.HandleFunc("/", service.handleWithData("", service.GetApps))
	for appID := range config.Apps {
		mux.HandleFunc(fmt.Sprintf("/%s", appID), service.handleWithData(appID, service.GetBundles))
		mux.HandleFunc(fmt.Sprintf("/%s/register", appID), service.handleWithData(appID, service.RegisterDevice))
		mux.HandleFunc(fmt.Sprintf("/%s/push", appID), service.handleWithData(appID, service.PushMessage))
	}
	return service, nil
}

// handleWithData принимает все запросы к сервису и отвечает за их разбор и вывод информации.
// Это промежуточный слой, выполняемый для всех запросов к сервису.
func (s *HTTPService) handleWithData(appID string, handle Handle) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST", "PUT": // поддерживаем только эти типы запросов
			// проверяем, что запрос в формате JSON
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
				return
			}
		case "GET": // тоже поддерживаем

		default: // остальные типы запросов не поддерживаются
			w.Header().Set("Allow", "POST,PUT")
			// http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// проверка авторизации приложений
		if appID != "" && len(s.config.Apps[appID].Keys) > 0 {
			auth := r.Header.Get("Authorization") // получаем из заголовка авторизационную информацию
			if auth == "" || !strings.HasPrefix(auth, tokenName) {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			auth = strings.TrimSpace(strings.TrimPrefix(auth, tokenName)) // избавляемся от префикса Token
			for _, appKey := range s.config.Apps[appID].Keys {
				if appKey == auth {
					goto Next
				}
			}
			// ни один ключ из конфигурации приложения не совпал
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

	Next:
		code, data := handle(appID, w, r) // вызываем оригинальный обработчик запроса
		switch code {
		case http.StatusOK, http.StatusInternalServerError, http.StatusBadRequest:
		default:
			code = http.StatusInternalServerError
		}
		if str, ok := data.(string); ok {
			data = map[string]interface{}{
				"code":   code,
				"status": str,
			}
		}
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			log.Println("Error:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; encoding=utf-8")
		w.WriteHeader(code)
		w.Write(jsonData)
		//log.Println("OK:", code, string(jsonData))
	}
}

// GetApps возвращает список поддерживаемых сервисом сервисов.
func (s *HTTPService) GetApps(_ string, w http.ResponseWriter, r *http.Request) (int, interface{}) {
	result := make([]string, 0, len(s.config.Apps))
	for app := range s.config.Apps {
		result = append(result, app)
	}
	return http.StatusOK, result
}

// GetBundles возвращает список приложений, поддерживаемых данным сервисом.
func (s *HTTPService) GetBundles(appID string, w http.ResponseWriter, r *http.Request) (int, interface{}) {
	result := make([]string, 0, len(s.config.Apps[appID].Bundles))
	for app := range s.config.Apps[appID].Bundles {
		result = append(result, app)
	}
	return http.StatusOK, result
}

// RegisterDevice регистрирует токен устройства в базе данных в привязке к сервису, пользователю и
// идентификатору приложения.
func (s *HTTPService) RegisterDevice(appID string, w http.ResponseWriter, r *http.Request) (int, interface{}) {
	// Разбираем параметры запроса
	var regDevice DeviceRegister
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&regDevice); err != nil {
		return http.StatusBadRequest, fmt.Sprintf("error parsing JSON request: %v", err)
	}
	// сохраняем в хранилище
	if err := s.store.AddDevice(appID, regDevice.Bundle, regDevice.User, regDevice.Token); err != nil {
		return http.StatusInternalServerError, err.Error()
	}
	return http.StatusOK, http.StatusText(http.StatusOK)
}

// PushMessage отправляет переданные push-уведомление на все устройства указанных в запросе
// пользователей.
func (s *HTTPService) PushMessage(appID string, w http.ResponseWriter, r *http.Request) (int, interface{}) {
	// разбираем параметры запроса
	var message PushMessage
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		return http.StatusBadRequest, fmt.Sprintf("error parsing JSON request: %v", err)
	}
	if len(message.Users) == 0 {
		return http.StatusBadRequest, errors.New("no users")
	}
	// получаем информацию о пользователях
	users, err := s.store.GetDevices(appID, message.Users...)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}
	if len(users) == 0 {
		return http.StatusOK, errors.New("no registered users")
	}
	// отсылаем push-уведомления
	for bundleID, push := range message.Messages {
		if push == nil {
			return http.StatusBadRequest, fmt.Errorf("empty message for %q", bundleID)
		}
		// получаем информацию о конфигурации для данного приложения
		var config = s.config.Apps[appID].Bundles[bundleID]
		if config == nil {
			return http.StatusBadRequest, fmt.Errorf("unknown bundle id %q", bundleID)
		}
		// собираем все токены от всех пользователей для данного приложения
		var tokens = make([]string, 0)
		for _, devices := range users {
			if toks := devices[bundleID]; len(toks) > 0 {
				tokens = append(tokens, toks...)
			}
		}
		if len(tokens) == 0 {
			log.Println("No tokens")
			continue // игнорируем отправку сообщений, когда некому посылать
		}
		switch config.Type {
		case "apns":
			// проверяем, что клиент для отправки определен
			if config.apnsClient == nil {
				return http.StatusInternalServerError, fmt.Errorf("client for %q not initialized", bundleID)
			}
			var notification = new(apns.Notification)
			if err = json.Unmarshal(push, notification); err != nil {
				return http.StatusBadRequest, fmt.Errorf("bad message format for %q: %s", bundleID, err)
			}
			if notification.Payload == nil || len(notification.Payload) == 0 {
				return http.StatusBadRequest, fmt.Errorf("bad empty message for %q", bundleID)
			}
			// отправляем сообщения
			log.Println("Push APNS to tokens:", strings.Join(tokens, ", "))
			if err := config.apnsClient.Send(notification, tokens...); err != nil {
				return http.StatusInternalServerError, err
			}
		case "gcm":
			if config.gcmClient == nil {
				return http.StatusInternalServerError, fmt.Errorf("client for %q not initialized", bundleID)
			}
			var msg = new(gcm.Message)
			if err = json.Unmarshal(push, msg); err != nil {
				return http.StatusBadRequest, fmt.Errorf("bad message format for %q: %s", bundleID, err)
			}
			if msg.Data == nil || len(msg.Data) == 0 {
				return http.StatusBadRequest, fmt.Errorf("bad empty message for %q", bundleID)
			}
			for len(tokens) > 0 {
				var toks []string
				if len(tokens) > 1000 {
					toks = tokens[:1000]
					tokens = tokens[1000:]
				} else {
					toks = tokens
					tokens = []string{}
				}
				msg.RegistrationIDs = toks
				response, err := config.gcmClient.Send(msg, 3)
				if err != nil {
					return http.StatusInternalServerError, err
				}
				_ = response
			}
		default:
			log.Printf("Ignore unknown %q type", bundleID)
			continue
		}
	}
	return http.StatusOK, http.StatusText(http.StatusOK)
}

// Close закрывает базу данных.
func (s *HTTPService) Close() error {
	return s.store.Close()
}
