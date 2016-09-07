package main

import (
	"sort"
	"sync"
)

// user описывает сохраняемую информацию о пользователе.
type user struct {
	Login    string
	Password hashPassword
}

// newUser возвращает инициализированное описание пользователя с хешом от пароля.
func newUser(login, password string) *user {
	return &user{login, newHashPassword(password)}
}

// Authorization описывает информацию для авторизации доступа к сервису.
type Authorization struct {
	administrator *user                   // авторизация администратора
	users         map[string]hashPassword // список пользователей и их паролей
	mu            sync.RWMutex
}

// Reset сбрасывает всех пользователей и администратора.
func (c *Authorization) Reset() {
	c.mu.Lock()
	c.administrator = nil
	c.users = nil
	c.mu.Unlock()
}

// SetAdmin устанавливает административную учетную запись сервиса. Возвращает
// true, если административная авторизация установлена.
func (c *Authorization) SetAdmin(login, password string) (secure bool) {
	c.mu.Lock()
	if login == "" {
		c.administrator = nil
	} else {
		c.administrator = newUser(login, password)
		secure = true
	}
	c.mu.Unlock()
	return secure
}

// AuthorizeAdmin возвращает true, если авторизация администратора совпадает
// с указанными параметрами.
func (c *Authorization) AuthorizeAdmin(login, password string) bool {
	c.mu.RLock()
	if c.administrator == nil || c.administrator.Login == "" {
		c.mu.RUnlock()
		return true
	}
	result := (login == c.administrator.Login &&
		c.administrator.Password.Equal(password))
	c.mu.RUnlock()
	return result
}

// IsAdminRequered возвращает true, если административная учетная запись
// установлена.
func (c *Authorization) IsAdminRequired() bool {
	c.mu.RLock()
	result := (c.administrator != nil && c.administrator.Login != "")
	c.mu.RUnlock()
	return result
}

// IsAuthorizationRequered возвращает true, если задан хотябы один пользователь
// для авторизации.
func (c *Authorization) IsAuthorizationRequired() bool {
	c.mu.RLock()
	result := (len(c.users) > 0)
	c.mu.RUnlock()
	return result
}

// IsUserExists возвращает true, если пользователь с таким логином
// зарегистрирован в списке пользователей.
func (c *Authorization) IsUserExists(login string) bool {
	c.mu.RLock()
	_, exists := c.users[login]
	c.mu.RUnlock()
	return exists
}

// AddUser добавляет пользователя в список авторизации пользователей.
// Возвращает true, если это новый пользователь. Если произошла замена пароля
// уже существующего пользователя, то возвращается false.
func (c *Authorization) AddUser(login, password string) (created bool) {
	if login == "" {
		return false
	}
	var exists bool
	c.mu.Lock()
	if c.users == nil {
		c.users = make(map[string]hashPassword)
	} else {
		_, exists = c.users[login]
	}
	c.users[login] = newHashPassword(password)
	c.mu.Unlock()
	return !exists
}

// RemoveUser удаляет пользователя с указанным логином из списка пользователей.
// Если пользователя с таким логином не существует в списке, то возвращается
// false.
func (c *Authorization) RemoveUser(login string) (exists bool) {
	c.mu.Lock()
	if c.users != nil {
		if _, exists = c.users[login]; exists {
			delete(c.users, login)
		}
	}
	c.mu.Unlock()
	return exists
}

// Authorize возвращает true, если авторизация пользователя прошла успешно.
// В том случае, если для авторизации не указан ни одни пользователь, любой
// логин и пароль является валидным.
func (c *Authorization) Authorize(login, password string) bool {
	c.mu.RLock()
	if len(c.users) == 0 {
		c.mu.RUnlock()
		return true
	}
	passwd, ok := c.users[login]
	c.mu.RUnlock()
	return (ok && passwd.Equal(password))
}

// Users возвращает список зарегистрированных пользователей.
func (c *Authorization) Users() []string {
	c.mu.RLock()
	list := make([]string, 0, len(c.users))
	for login := range c.users {
		list = append(list, login)
	}
	c.mu.RUnlock()
	sort.Strings(list)
	return list
}

// getUsers возвращает список пользователей с паролями.
func (c *Authorization) getUsers() []user {
	c.mu.RLock()
	list := make([]user, 0, len(c.users))
	for login, passwd := range c.users {
		list = append(list, user{login, passwd})
	}
	c.mu.RUnlock()
	return list
}

// restoreUsers восстанавливает список пользователей из сохраненного формата.
func restoreUsers(list []user) map[string]hashPassword {
	users := make(map[string]hashPassword, len(list))
	for _, user := range list {
		users[user.Login] = user.Password
	}
	return users
}
