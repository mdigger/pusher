package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mdigger/log"
	"github.com/mdigger/rest3"
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
			"Server":            "Pusher/2.0",
			"X-API-Version":     "1.1",
			"X-Service-Version": version,
		},
		Logger: log.Default,
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
func (s *Service) AdminAuth(c *rest.Context) error {
	if !s.config.IsAdminAuthorization() {
		return nil // авторизация не требуется
	}
	// разбираем заголовок с авторизацией
	login, password, ok := c.BasicAuth()
	if !ok {
		realm := fmt.Sprintf("Basic realm=%s admin", appName)
		c.SetHeader("WWW-Authenticate", realm)
		return rest.ErrUnauthorized
	}
	// проверяем авторизацию
	if !s.config.AdminAuthorization(login, password) {
		return rest.ErrForbidden
	}
	return nil // администратор авторизован
}

// GetUsers отдает список пользователей для авторизации.
func (s *Service) GetUsers(c *rest.Context) error {
	// проверяем авторизацию администратора
	if err := s.AdminAuth(c); err != nil {
		return err
	}
	// отдаем список пользователей
	return c.Write(rest.JSON{"users": s.config.UsersList()})
}

// AddUser регистрирует нового пользователя для авторизации обращения к сервису.
func (s *Service) AddUser(c *rest.Context) error {
	// проверяем авторизацию администратора
	if err := s.AdminAuth(c); err != nil {
		return err
	}
	// разбираем информацию о пользователе из запроса
	var user = new(struct {
		Login    string `json:"login" form:"login"`
		Password string `json:"password" form:"password"`
	})
	err := c.Bind(user)
	if err != nil {
		return err
	}
	if user.Login == "" {
		return rest.ErrBadRequest
	}
	// добавляем информацию о пользователе
	exist := s.config.AddUser(user.Login, user.Password)
	// если это новый пользователь, то отдаем статус создания
	var code = http.StatusOK
	if !exist {
		code = http.StatusCreated
	}
	c.SetStatus(code)
	// отдаем список пользователей
	return c.Write(rest.JSON{"users": s.config.UsersList()})
}

// RemoveUser удаляет пользователя из списка авторизации.
func (s *Service) RemoveUser(c *rest.Context) error {
	// проверяем авторизацию администратора
	if err := s.AdminAuth(c); err != nil {
		return err
	}
	// получаем логин из запроса пути
	var login = c.Param("login")
	// удаляем пользователя из списка
	exist := s.config.RemoveUser(login)
	if !exist {
		return c.Error(http.StatusNotFound, fmt.Sprintf("user %s not registered", login))
	}
	// отдаем список пользователей
	return c.Write(rest.JSON{"users": s.config.UsersList()})
}

// ChangeUser изменяет пароль пользователя для авторизации.
func (s *Service) ChangeUser(c *rest.Context) error {
	// проверяем авторизацию администратора
	if err := s.AdminAuth(c); err != nil {
		return err
	}
	// получаем логин из запроса пути
	var login = c.Param("login")
	// получаем пароль из запроса
	var password = new(struct {
		Password string `json:"password" form:"password"`
	})
	err := c.Bind(password)
	if err != nil {
		return err
	}
	// изменяем пароль пользователя
	exist := s.config.AddUser(login, password.Password)
	code := http.StatusOK
	if !exist {
		code = http.StatusCreated
	}
	c.SetStatus(code)
	// отдаем список пользователей
	return c.Write(rest.JSON{"users": s.config.UsersList()})
}

// UserAuth проверяет авторизацию пользователя. Возвращает 0, nil, если
// пользователь успешно авторизован или авторизация не требуется.
func (s *Service) UserAuth(c *rest.Context) error {
	if !s.config.IsUserAuthorization() {
		return nil // авторизация не требуется
	}
	// разбираем заголовок с авторизацией
	login, password, ok := c.BasicAuth()
	if !ok {
		realm := fmt.Sprintf("Basic realm=%s", appName)
		c.SetHeader("WWW-Authenticate", realm)
		return rest.ErrUnauthorized
	}
	// проверяем авторизацию
	if !s.config.UserAuthorization(login, password) {
		return rest.ErrForbidden
	}
	return nil // администратор авторизован
}

// GetTokens отдает список зарегистрированных токенов пользователя
func (s *Service) GetTokens(c *rest.Context) error {
	// проверяем авторизацию пользователя
	if err := s.UserAuth(c); err != nil {
		return err
	}
	user := c.Param("login") // логин пользователя
	if user == "" {
		return rest.ErrNotFound
	}
	topic := c.Param("topic") // тема
	if topic == "" {
		return rest.ErrNotFound
	}
	query := c.Request.URL.Query()        // разобранные параметры запроса
	sandbox := len(query["sandbox"]) != 0 // флаг sandbox
	// запрашиваем список токенов пользователя
	tokens, err := s.config.Store.GetUserTopicTokens(topic, sandbox, user)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return c.Error(http.StatusNotFound,
			fmt.Sprintf("tokens for user %s not registered", user))
	}
	return c.Write(rest.JSON{"tokens": tokens})
}

// AddToken регистрирует токен пользовательского устройства.
func (s *Service) AddToken(c *rest.Context) error {
	// проверяем авторизацию пользователя
	if err := s.UserAuth(c); err != nil {
		return err
	}
	user := c.Param("login") // логин пользователя
	if user == "" {
		return rest.ErrNotFound
	}
	topic := c.Param("topic") // тема
	if topic == "" {
		return c.Error(http.StatusNotFound, "empty topic")
	}
	query := c.Request.URL.Query()        // разобранные параметры запроса
	sandbox := len(query["sandbox"]) != 0 // флаг sandbox
	var token = new(struct {
		Token string `json:"token" form:"token"`
	})
	err := c.Bind(token)
	if err != nil {
		return err
	}
	// сохраняем в хранилище токенов устройств
	err = s.config.AddToken(user, topic, token.Token, sandbox)
	if err != nil {
		return err
	}
	// запрашиваем список токенов пользователя
	tokens, err := s.config.Store.GetUserTopicTokens(topic, sandbox, user)
	if err != nil {
		return err
	}
	c.SetStatus(http.StatusCreated)
	return c.Write(rest.JSON{"tokens": tokens})
}

// PushUser отправляет push-уведомления на все устройства пользователя.
func (s *Service) PushUser(c *rest.Context) error {
	// проверяем авторизацию пользователя
	if err := s.UserAuth(c); err != nil {
		return err
	}
	user := c.Param("login") // логин пользователя
	if user == "" {
		return rest.ErrNotFound
	}
	topic := c.Param("topic") // тема
	if topic == "" {
		return c.Error(http.StatusNotFound, "empty topic")
	}
	query := c.Request.URL.Query()        // разобранные параметры запроса
	sandbox := len(query["sandbox"]) != 0 // флаг sandbox
	// получаем список токенов пользователя
	tokens, err := s.config.Store.GetUserTopicTokens(topic, sandbox, user)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return c.Error(http.StatusNotFound,
			fmt.Sprintf("tokens for user %s not registered", user))
	}
	// разбираем запроса для отправки уведомления
	var notification = new(struct {
		Payload     map[string]interface{} `json:"payload" form:"payload"`
		Expiration  time.Time              `json:"expiration" form:"expiration"`
		LowPriority bool                   `json:"lowPriority" form:"lowPriority"`
		CollapseID  string                 `json:"collapseId" form:"collapseId"`
	})
	err = c.Bind(notification)
	if err != nil {
		return err
	}
	if len(notification.Payload) == 0 {
		return c.Error(http.StatusBadRequest, "empty payload")
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
		return err
	}
	// отдаем количество отправленных сообщений
	return c.Write(rest.JSON{"sent": sent})
}

// Push отправляет push-уведомления на все устройства указанных в запросе
// пользователей.
func (s *Service) Push(c *rest.Context) error {
	// проверяем авторизацию пользователя
	if err := s.UserAuth(c); err != nil {
		return err
	}
	topic := c.Param("topic") // тема
	if topic == "" {
		return c.Error(http.StatusNotFound, "empty topic")
	}
	query := c.Request.URL.Query()        // разобранные параметры запроса
	sandbox := len(query["sandbox"]) != 0 // флаг sandbox
	// разбираем запроса для отправки уведомления
	var notification = new(struct {
		Payload     map[string]interface{} `json:"payload" form:"payload"`
		Expiration  time.Time              `json:"expiration" form:"expiration"`
		LowPriority bool                   `json:"lowPriority" form:"lowPriority"`
		CollapseID  string                 `json:"collapseId" form:"collapseId"`
		Users       []string               `json:"users" form:"user"`
	})
	err := c.Bind(notification)
	if err != nil {
		return err
	}

	if len(notification.Payload) == 0 {
		return c.Error(http.StatusBadRequest, "empty payload")
	}
	if len(notification.Users) == 0 {
		return c.Error(http.StatusBadRequest, "empty users list")
	}
	// получаем список токенов пользователя
	tokens, err := s.config.Store.GetUserTopicTokens(topic, sandbox,
		notification.Users...)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return c.Error(http.StatusNotFound, "tokens not registered")
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
		return err
	}
	// отдаем количество отправленных сообщений
	return c.Write(rest.JSON{"sent": sent})
}
