package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/mdigger/log"
)

// Password описывает хеш от пароля для хранения.
type Password []byte

// NewPassword возвращает хеш от пароля
func NewPassword(password string) Password {
	var sum = sha256.Sum224([]byte(password))
	return sum[:]
}

// Equal сравнивает указанный пароль с сохраненным и возвращает true, если они
// полностью совпадают.
func (p Password) Equal(password string) bool {
	var passwd = NewPassword(password)
	return bytes.Equal(p, passwd)
}

// Admin описывает данные для авторизации администратора.
type Admin struct {
	Login    string   `json:"login"`
	Password Password `json:"password"`
}

// Users содержит список пользователей для авторизации.
type Users map[string]Password

// Config описывает конфигурацию сервиса.
type Config struct {
	Admin    *Admin         `json:"admin,omitempty"`
	Users    Users          `json:"users,omitempty"`
	Provider *ProviderToken `json:"apnsToken,omitempty"`
	Store    *Store         `json:"deviceTokens,omitempty"`
	mu       sync.RWMutex
}

// LoadConfig загружает конфигурацию сервиса из файла.
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	var service = new(Config)
	err = json.NewDecoder(file).Decode(service)
	file.Close()
	if err != nil {
		return nil, err
	}
	// инициализируем хранилище, если оно не указано в конфигурации
	if service.Store == nil {
		store, err := OpenStore("tokens.db")
		if err != nil {
			return nil, err
		}
		service.Store = store
	}
	return service, nil
}

// Save сохраняет конфигурацию в файл.
func (c *Config) Save() error {
	file, err := os.Create(config)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	err = enc.Encode(c)
	file.Close()
	return err
}

// Close закрывает сервис.
func (c *Config) Close() error {
	if c.Store != nil {
		return c.Store.Close()
	}
	return nil
}

// SetAdmin устанавливает логин и пароль для авторизации администратора. Если
// логин пустой, то авторизация администратора не требуется. Возвращает true,
// если авторизация для администратора установлена.
func (c *Config) SetAdmin(login, password string) (secure bool) {
	c.mu.Lock()
	if login == "" {
		c.Admin = nil
		secure = false
		log.Debug("clear admin")
	} else {
		c.Admin = &Admin{
			Login:    login,
			Password: NewPassword(password),
		}
		secure = true
		log.WithField("login", login).Debug("set admin")
	}
	c.mu.Unlock()
	return
}

// IsAdminAuthorization возвращает true, если требуется авторизация
// администратора.
func (c *Config) IsAdminAuthorization() bool {
	c.mu.RLock()
	var result = c.Admin != nil && c.Admin.Login != ""
	c.mu.RUnlock()
	return result
}

// AdminAuthorization возвращает true, если авторизация администратора совпадает
// с заданной в конфигурации или авторизация администратора не требуется.
func (c *Config) AdminAuthorization(login, password string) (ok bool) {
	c.mu.RLock()
	if c.Admin == nil || c.Admin.Login == "" {
		c.mu.RUnlock()
		return true // авторизация не задана — подходит любая
	}
	// иначе сравниваем логин и пароль с заданным в конфигурации
	ok = c.Admin.Login == login && c.Admin.Password.Equal(password)
	c.mu.RUnlock()
	return
}

// IsUserAuthorization возвращает true, если требуется авторизация пользователя.
func (c *Config) IsUserAuthorization() bool {
	c.mu.RLock()
	var result = len(c.Users) > 0
	c.mu.RUnlock()
	return result
}

// UserAuthorization возвращает true если пользователь с заданными логином и
// паролем существует в списке авторизованных пользователей или авторизация для
// сервиса не задана.
func (c *Config) UserAuthorization(login, password string) bool {
	c.mu.RLock()
	if len(c.Users) == 0 {
		c.mu.RUnlock()
		return true // авторизация не задана — подходит любая
	}
	// иначе сравниваем логин и пароль с заданным в конфигурации
	passwd, exist := c.Users[login]
	c.mu.RUnlock()
	return exist && passwd.Equal(password)
}

// AddUser добавляет нового пользователя для авторизации. Возвращает true, если
// пользователь с таким логином уже был зарегистрирован и произошла замена
// пароля для него. Если это новый пользователь, то возвращается false.
func (c *Config) AddUser(login, password string) (exist bool) {
	log.WithField("login", login).Debug("add user")
	c.mu.Lock()
	if c.Users == nil {
		c.Users = make(Users)
	} else {
		_, exist = c.Users[login]
	}
	c.Users[login] = NewPassword(password)
	c.mu.Unlock()
	return
}

// RemoveUser удаляет пользователя из списка зарегистрированных. Возвращает
// true, если пользователь с таким логином был зарегистрирован ранее.
func (c *Config) RemoveUser(login string) (exist bool) {
	ctxlog := log.WithField("login", login)
	c.mu.Lock()
	if c.Users == nil {
		c.mu.Unlock()
		ctxlog.Warning("remove user: empty users list")
		return false
	}
	_, exist = c.Users[login]
	if exist {
		ctxlog.Debug("remove user")
		delete(c.Users, login)
	} else {
		ctxlog.Warning("remove user: not found")
	}
	c.mu.Unlock()
	return
}

// UsersList возвращает список зарегистрированных пользователей.
func (c *Config) UsersList() []string {
	c.mu.RLock()
	var list = make([]string, 0, len(c.Users))
	for login := range c.Users {
		list = append(list, login)
	}
	c.mu.RUnlock()
	sort.Strings(list)
	return list
}

// SetProviderToken устанавливает APNS токен для конфигурации.
func (c *Config) SetProviderToken(teamID, keyID string, privateKeyData []byte) error {
	log.WithFields(log.Fields{
		"team":    teamID,
		"key":     keyID,
		"private": len(privateKeyData),
	}).Debug("set token")
	pt, err := NewProviderToken(teamID, keyID, privateKeyData)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.Provider = pt
	c.mu.Unlock()
	return nil
}

// AddToken добавляет токен устройства пользователя в хранилище токенов.
func (c *Config) AddToken(user, topic, token string, sandbox bool) error {
	ctxlog := log.WithFields(log.Fields{
		"user":    user,
		"topic":   topic,
		"token":   token,
		"sandbox": sandbox,
	})
	err := c.Store.Save(user, topic, token, sandbox)
	if err != nil {
		ctxlog.WithError(err).Error("add token error")
	} else {
		ctxlog.Debug("add user token")
	}
	return err
}

// RemoveToken удаляет токен из хранилища.
func (c *Config) RemoveToken(topic, token string, timestamp time.Time, sandbox bool) error {
	ctxlog := log.WithFields(log.Fields{
		"topic":   topic,
		"token":   token,
		"sandbox": sandbox,
	})
	if !timestamp.IsZero() {
		ctxlog = ctxlog.WithField("timestamp", timestamp)
	}
	err := c.Store.Remove(topic, token, timestamp, sandbox)
	if err != nil {
		ctxlog.WithError(err).Error("remove token error")
	} else {
		ctxlog.Debug("remove token")
	}
	return err
}

// Push отправляет push-уведомление на сервер APNS.
func (c *Config) Push(notification Notification, tokens []string) (
	status map[string]string, err error) {
	status = make(map[string]string, len(tokens))
	for _, token := range tokens {
		ctxlog := log.WithFields(log.Fields{
			"token": token,
			"topic": notification.Topic,
		})
		notification.Token = token
		_, err = c.Provider.Push(notification)
		if err == nil {
			ctxlog.Debug("push sent")
			status[token] = "OK"
			continue
		}
		if apnserr, ok := err.(*Error); ok {
			ctxlog = ctxlog.WithError(err).WithFields(log.Fields{
				"reason":  apnserr.Reason,
				"status":  apnserr.Status,
				"isToken": apnserr.IsToken(),
			})
			status[token] = apnserr.Error()
			if apnserr.IsToken() {
				ctxlog.Warning("token error")
				// удаляем токен в случае ошибки связанной с ним
				err = c.RemoveToken(token,
					notification.Topic,
					apnserr.Time(),
					notification.Sandbox)
				if err == nil {
					continue // переходим к следующему токену
				}
			}
		} else {
			status[token] = err.Error()
		}
		ctxlog.WithError(err).Error("push error")
		return status, err
	}
	return status, nil
}
