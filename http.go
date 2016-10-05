package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mdigger/log"
	"github.com/mdigger/rest"
)

type Service struct {
	mux    *rest.ServeMux // мультиплексор запросов
	config *Config        // конфигурация сервиса
}

// NewService инициализирует новый сервис по конфигурации.
func NewService(config *Config) *Service {
	// инициализируем мультиплексор HTTP-запросов
	var mux = &rest.ServeMux{
		Headers: map[string]string{
			"X-API-Version":     "1.1",
			"X-Service-Version": version,
		},
		Logger:     log.Default,
		SendErrors: true,
		Options: &rest.Options{
			DataAdapter: rest.Adapter,
			Encoder:     rest.JSONEncoder(true),
		},
	}
	var service = &Service{
		mux:    mux,
		config: config,
	}
	// добавляем обработчики запросов администрирования
	mux.Handle("GET", "/users", service.GetUsers)
	mux.Handle("POST", "/users", service.AddUser)
	mux.Handle("DELETE", "/users/:login", service.RemoveUser)
	mux.Handle("PUT", "/users/:login", service.ChangeUser)
	// токены устройств пользователя
	mux.Handle("GET", "/apns/:topic/users/:login", service.GetTokens)
	mux.Handle("POST", "/apns/:topic/users/:login", service.AddToken)
	// отправка push-уведомлений
	mux.Handle("POST", "/apns/:topic/push", service.Push)
	mux.Handle("POST", "/apns/:topic/users/:login/push", service.PushUser)
	return service
}

// AdminAuth проверяет авторизацию администратора. Возвращает 0, nil, если
// администратор успешно авторизован.
func (s *Service) AdminAuth(w http.ResponseWriter, r *http.Request) (int, error) {
	if !s.config.IsAdminAuthorization() {
		return 0, nil // авторизация не требуется
	}
	// разбираем заголовок с авторизацией
	login, password, ok := r.BasicAuth()
	if !ok {
		realm := fmt.Sprintf("Basic realm=%s admin", appName)
		w.Header().Set("WWW-Authenticate", realm)
		return http.StatusUnauthorized, errors.New("no admin authorization")
	}
	// проверяем авторизацию
	if !s.config.AdminAuthorization(login, password) {
		return http.StatusForbidden, errors.New("bad admin authorization")
	}
	return 0, nil // администратор авторизован
}

// GetUsers отдает список пользователей для авторизации.
func (s *Service) GetUsers(w http.ResponseWriter, r *http.Request) (int, error) {
	// проверяем авторизацию администратора
	if code, err := s.AdminAuth(w, r); code != 0 || err != nil {
		return code, err
	}
	// отдаем список пользователей
	return rest.Write(w, r, 200, rest.JSON{"users": s.config.UsersList()})
}

// AddUser регистрирует нового пользователя для авторизации обращения к сервису.
func (s *Service) AddUser(w http.ResponseWriter, r *http.Request) (int, error) {
	// проверяем авторизацию администратора
	if code, err := s.AdminAuth(w, r); code != 0 || err != nil {
		return code, err
	}
	// разбираем информацию о пользователе из запроса
	var user = new(struct {
		Login    string `json:"login" form:"login"`
		Password string `json:"password" form:"password"`
	})
	err := rest.Bind(r, user)
	if err != nil {
		return http.StatusBadRequest, err
	}
	if user.Login == "" {
		return http.StatusBadRequest, errors.New("empty login")
	}
	// добавляем информацию о пользователе
	exist := s.config.AddUser(user.Login, user.Password)
	// если это новый пользователь, то отдаем статус создания
	var code = http.StatusOK
	if !exist {
		code = http.StatusCreated
	}
	// отдаем список пользователей
	return rest.Write(w, r, code, rest.JSON{"users": s.config.UsersList()})
}

// RemoveUser удаляет пользователя из списка авторизации.
func (s *Service) RemoveUser(w http.ResponseWriter, r *http.Request) (int, error) {
	// проверяем авторизацию администратора
	if code, err := s.AdminAuth(w, r); code != 0 || err != nil {
		return code, err
	}
	// получаем логин из запроса пути
	var login = rest.Params(r).Get("login")
	// удаляем пользователя из списка
	exist := s.config.RemoveUser(login)
	if !exist {
		return http.StatusNotFound, fmt.Errorf("user %s not registered", login)
	}
	// отдаем список пользователей
	return rest.Write(w, r, http.StatusOK, rest.JSON{"users": s.config.UsersList()})
}

// ChangeUser изменяет пароль пользователя для авторизации.
func (s *Service) ChangeUser(w http.ResponseWriter, r *http.Request) (int, error) {
	// проверяем авторизацию администратора
	if code, err := s.AdminAuth(w, r); code != 0 || err != nil {
		return code, err
	}
	// получаем логин из запроса пути
	var login = rest.Params(r).Get("login")
	// получаем пароль из запроса
	var password = new(struct {
		Password string `json:"password" form:"password"`
	})
	err := rest.Bind(r, password)
	if err != nil {
		return http.StatusBadRequest, err
	}
	// изменяем пароль пользователя
	exist := s.config.AddUser(login, password.Password)
	code := http.StatusOK
	if !exist {
		code = http.StatusCreated
	}
	// отдаем список пользователей
	return rest.Write(w, r, code, rest.JSON{"users": s.config.UsersList()})
}

// UserAuth проверяет авторизацию пользователя. Возвращает 0, nil, если
// пользователь успешно авторизован или авторизация не требуется.
func (s *Service) UserAuth(w http.ResponseWriter, r *http.Request) (int, error) {
	if !s.config.IsUserAuthorization() {
		return 0, nil // авторизация не требуется
	}
	// разбираем заголовок с авторизацией
	login, password, ok := r.BasicAuth()
	if !ok {
		realm := fmt.Sprintf("Basic realm=%s", appName)
		w.Header().Set("WWW-Authenticate", realm)
		return http.StatusUnauthorized, errors.New("no user authorization")
	}
	// проверяем авторизацию
	if !s.config.UserAuthorization(login, password) {
		return http.StatusForbidden, errors.New("bad user authorization")
	}
	return 0, nil // администратор авторизован
}

// GetTokens отдает список зарегистрированных токенов пользователя
func (s *Service) GetTokens(w http.ResponseWriter, r *http.Request) (int, error) {
	// проверяем авторизацию пользователя
	if code, err := s.UserAuth(w, r); code != 0 || err != nil {
		return code, err
	}
	params := rest.Params(r)    // параметры пути
	user := params.Get("login") // логин пользователя
	if user == "" {
		return http.StatusNotFound, errors.New("empty user login")
	}
	topic := params.Get("topic") // тема
	if topic == "" {
		return http.StatusNotFound, errors.New("empty topic")
	}
	query := r.URL.Query()                // разобранные параметры запроса
	sandbox := len(query["sandbox"]) != 0 // флаг sandbox
	// запрашиваем список токенов пользователя
	tokens, err := s.config.Store.GetUserTopicTokens(topic, sandbox, user)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if len(tokens) == 0 {
		return http.StatusNotFound,
			fmt.Errorf("tokens for user %s not registered", user)
	}
	return rest.Write(w, r, http.StatusOK, rest.JSON{"tokens": tokens})
}

// AddToken регистрирует токен пользовательского устройства.
func (s *Service) AddToken(w http.ResponseWriter, r *http.Request) (int, error) {
	// проверяем авторизацию пользователя
	if code, err := s.UserAuth(w, r); code != 0 || err != nil {
		return code, err
	}
	params := rest.Params(r)    // параметры пути
	user := params.Get("login") // логин пользователя
	if user == "" {
		return http.StatusNotFound, errors.New("empty user login")
	}
	topic := params.Get("topic") // тема
	if topic == "" {
		return http.StatusNotFound, errors.New("empty topic")
	}
	query := r.URL.Query()                // разобранные параметры запроса
	sandbox := len(query["sandbox"]) != 0 // флаг sandbox
	var token = new(struct {
		Token string `json:"token" form:"token"`
	})
	err := rest.Bind(r, token)
	if err != nil {
		return http.StatusBadRequest, err
	}
	// сохраняем в хранилище токенов устройств
	err = s.config.AddToken(user, topic, token.Token, sandbox)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	// запрашиваем список токенов пользователя
	tokens, err := s.config.Store.GetUserTopicTokens(topic, sandbox, user)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return rest.Write(w, r, http.StatusCreated, rest.JSON{"tokens": tokens})
}

// PushUser отправляет push-уведомления на все устройства пользователя.
func (s *Service) PushUser(w http.ResponseWriter, r *http.Request) (int, error) {
	// проверяем авторизацию пользователя
	if code, err := s.UserAuth(w, r); code != 0 || err != nil {
		return code, err
	}
	params := rest.Params(r)    // параметры пути
	user := params.Get("login") // логин пользователя
	if user == "" {
		return http.StatusNotFound, errors.New("empty user login")
	}
	topic := params.Get("topic") // тема
	if topic == "" {
		return http.StatusNotFound, errors.New("empty topic")
	}
	query := r.URL.Query()                // разобранные параметры запроса
	sandbox := len(query["sandbox"]) != 0 // флаг sandbox
	// получаем список токенов пользователя
	tokens, err := s.config.Store.GetUserTopicTokens(topic, sandbox, user)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if len(tokens) == 0 {
		return http.StatusNotFound,
			fmt.Errorf("tokens for user %s not registered", user)
	}
	// разбираем запроса для отправки уведомления
	var notification = new(struct {
		Payload     map[string]interface{} `json:"payload" form:"payload"`
		Expiration  time.Time              `json:"expiration" form:"expiration"`
		LowPriority bool                   `json:"lowPriority" form:"lowPriority"`
		CollapseID  string                 `json:"collapseId" form:"collapseId"`
	})
	err = rest.Bind(r, notification)
	if err != nil {
		return http.StatusBadRequest, err
	}
	if len(notification.Payload) == 0 {
		return http.StatusBadRequest, errors.New("empty payload")
	}
	// формируем данные для уведомления
	var n = Notification{
		Payload:     notification.Payload,
		Expiration:  notification.Expiration,
		LowPriority: notification.LowPriority,
		Topic:       topic,
		CollapseID:  notification.CollapseID,
		Sandbox:     sandbox,
	}
	// отправляем на все токены пользователя
	// отправляем на все токены пользователя
	sent, err := s.config.Push(n, tokens)
	if err != nil {
		return http.StatusBadRequest, err
	}
	// отдаем количество отправленных сообщений
	return rest.Write(w, r, http.StatusOK, rest.JSON{"sent": sent})
}

// Push отправляет push-уведомления на все устройства указанных в запросе
// пользователей.
func (s *Service) Push(w http.ResponseWriter, r *http.Request) (int, error) {
	// проверяем авторизацию пользователя
	if code, err := s.UserAuth(w, r); code != 0 || err != nil {
		return code, err
	}
	params := rest.Params(r)     // параметры пути
	topic := params.Get("topic") // тема
	if topic == "" {
		return http.StatusNotFound, errors.New("empty topic")
	}
	query := r.URL.Query()                // разобранные параметры запроса
	sandbox := len(query["sandbox"]) != 0 // флаг sandbox
	// разбираем запроса для отправки уведомления
	var notification = new(struct {
		Payload     map[string]interface{} `json:"payload" form:"payload"`
		Expiration  time.Time              `json:"expiration" form:"expiration"`
		LowPriority bool                   `json:"lowPriority" form:"lowPriority"`
		CollapseID  string                 `json:"collapseId" form:"collapseId"`
		Users       []string               `json:"users" form:"user"`
	})
	err := rest.Bind(r, notification)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if len(notification.Payload) == 0 {
		return http.StatusBadRequest, errors.New("empty payload")
	}
	if len(notification.Users) == 0 {
		return http.StatusBadRequest, errors.New("empty users list")
	}
	// получаем список токенов пользователя
	tokens, err := s.config.Store.GetUserTopicTokens(topic, sandbox,
		notification.Users...)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if len(tokens) == 0 {
		return http.StatusNotFound, errors.New("tokens not registered")
	}
	// формируем данные для уведомления
	var n = Notification{
		Payload:     notification.Payload,
		Expiration:  notification.Expiration,
		LowPriority: notification.LowPriority,
		Topic:       topic,
		CollapseID:  notification.CollapseID,
		Sandbox:     sandbox,
	}
	// отправляем на все токены пользователя
	sent, err := s.config.Push(n, tokens)
	if err != nil {
		return http.StatusBadRequest, err
	}
	// отдаем количество отправленных сообщений
	return rest.Write(w, r, http.StatusOK, rest.JSON{"sent": sent})
}
