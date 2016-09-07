package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"time"

	"golang.org/x/crypto/pkcs12"

	"github.com/mdigger/apns3"
	"github.com/mdigger/rest"
)

// registerHandlers регистрирует обработчик HTTP-запросов сервиса.
func (cfg *Config) registerHandlers(mux *rest.ServeMux) {
	// административная часть, связанная с сертификатами, пользователями и
	// администратором.
	mux.Handles(rest.Paths{
		"/certificates": {
			// возвращает список идентификаторов сертификатов
			"GET": func(c *rest.Context) error {
				return c.Send(rest.JSON{"certificates": cfg.Certificates()})
			},
			// регистрирует новый сертификат
			"POST": func(c *rest.Context) error {
				var certificate struct {
					Certificate []byte // содержимое сертификата
					Password    string // пароль для его чтения
					Sandbox     bool   // флаг тестового подключения
				}
				// разбираем заголовок с типом информации в запросе
				mediatype, _, _ := mime.ParseMediaType(
					c.Request.Header.Get("Content-Type"))
				// в зависимости от формата запроса, разбираем данные
				if mediatype == "application/json" {
					if err := c.Bind(&certificate); err != nil {
						return c.Send(err)
					}
				} else {
					certificate.Password = c.PostFormValue("password")
					certificate.Sandbox = (c.PostFormValue("sandbox") != "")
					if mediatype == "application/x-www-form-urlencoded" {
						cert := c.PostFormValue("certificate")
						data, err := base64.StdEncoding.DecodeString(cert)
						if err != nil {
							return c.Error(http.StatusBadRequest, err.Error())
						}
						certificate.Certificate = data
					} else if mediatype == "multipart/form-data" {
						file, _, err := c.FormFile("certificate")
						if err != nil {
							return c.Error(http.StatusBadRequest, err.Error())
						}
						data, err := ioutil.ReadAll(file)
						file.Close()
						if err != nil {
							return c.Error(http.StatusBadRequest, err.Error())
						}
						certificate.Certificate = data
					} else {
						return c.Send(rest.ErrUnsupportedMediaType)
					}
				}
				// разбираем сертификат
				privateKey, x509Cert, err := pkcs12.Decode(
					certificate.Certificate, certificate.Password)
				if err != nil {
					return c.Error(http.StatusBadRequest, err.Error())
				}
				cert := &tls.Certificate{
					Certificate: [][]byte{x509Cert.Raw},
					PrivateKey:  privateKey,
					Leaf:        x509Cert,
				}
				if _, err = x509Cert.Verify(x509.VerifyOptions{}); err != nil {
					if _, ok := err.(x509.UnknownAuthorityError); !ok {
						return c.Error(http.StatusBadRequest, err.Error())
					}
				}
				// добавляем его в список сертификатов
				info, replaced, err := cfg.AddCertificate(*cert, certificate.Sandbox)
				if err != nil {
					return c.Error(http.StatusBadRequest, err.Error())
				}
				// сохраняем конфигурацию
				if err := cfg.Save(); err != nil {
					return err
				}
				if !replaced {
					c.Status(http.StatusCreated)
				}
				c.Header().Set("Location",
					fmt.Sprintf("/certificates/%s", info.BundleID))
				// возвращаем информацию о сертификате
				return c.Send(info)
			},
		},
		"/certificates/:id": {
			// возвращает информацию о сертификате
			"GET": func(c *rest.Context) error {
				id := c.Param("id")
				if client := cfg.Client(id); client != nil {
					return c.Send(client.CertificateInfo)
				}
				return c.Send(rest.ErrNotFound)
			},
			// удаляет сертификат и всех связанных с ним клиентов из списка
			// зарегистрированных сертификатов
			"DELETE": func(c *rest.Context) error {
				if !cfg.Remove(c.Param("id")) {
					return c.Send(rest.ErrNotFound)
				}
				// сохраняем конфигурацию
				if err := cfg.Save(); err != nil {
					return err
				}
				return c.Send(nil)
			},
		},
		"/users": {
			// возвращает список идентификаторов сертификатов
			"GET": func(c *rest.Context) error {
				return c.Send(rest.JSON{"users": cfg.Users()})
			},
			// регистрирует нового пользователя
			"POST": func(c *rest.Context) error {
				var user struct{ Login, Password string }
				if err := c.Bind(&user); err == rest.ErrUnsupportedMediaType {
					user.Login = c.PostFormValue("login")
					user.Password = c.PostFormValue("password")
				} else if err != nil {
					return c.Send(err)
				}
				if user.Login == "" {
					return c.Error(http.StatusBadRequest, "User login not set")
				}
				if cfg.AddUser(user.Login, user.Password) {
					c.Status(http.StatusCreated)
				}
				// сохраняем конфигурацию
				if err := cfg.Save(); err != nil {
					return err
				}
				c.Header().Set("Location", fmt.Sprintf("/users/%s", user.Login))
				return c.Send(nil)
			},
		},
		"/users/:name": {
			// возвращает ошибку, если пользователь не зарегистрирован
			"GET": func(c *rest.Context) error {
				if !cfg.IsUserExists(c.Param("name")) {
					return c.Send(rest.ErrNotFound)
				}
				return c.Send(nil)
			},
			// устанавливает пароль для пользователя
			// создает нового пользователя, если раньше его не существовало
			"PUT": func(c *rest.Context) error {
				var user struct{ Password string }
				if err := c.Bind(&user); err == rest.ErrUnsupportedMediaType {
					user.Password = c.PostFormValue("password")
				} else if err != nil {
					return c.Send(err)
				}
				if cfg.AddUser(c.Param("name"), user.Password) {
					c.Status(http.StatusCreated)
				}
				// сохраняем конфигурацию
				if err := cfg.Save(); err != nil {
					return err
				}
				return c.Send(nil)
			},
			// удаляет пользователя из списка авторизации
			"DELETE": func(c *rest.Context) error {
				if !cfg.RemoveUser(c.Param("name")) {
					return c.Send(rest.ErrNotFound)
				}
				// сохраняем конфигурацию
				if err := cfg.Save(); err != nil {
					return err
				}
				return c.Send(nil)
			},
		},
		"/admin": {
			// возвращает true, если администратор определен
			"GET": func(c *rest.Context) error {
				return c.Send(rest.JSON{"secured": cfg.IsAdminRequired()})
			},
			// изменяет логин и пароль администратора
			"POST": func(c *rest.Context) error {
				var user struct{ Login, Password string }
				if err := c.Bind(&user); err == rest.ErrUnsupportedMediaType {
					user.Login = c.PostFormValue("login")
					user.Password = c.PostFormValue("password")
				} else if err != nil {
					return c.Send(err)
				}
				// сохраняем конфигурацию
				if err := cfg.Save(); err != nil {
					return err
				}
				return c.Send(rest.JSON{
					"secured": cfg.SetAdmin(user.Login, user.Password)})
			},
			// удаляет администратора
			"DELETE": func(c *rest.Context) error {
				cfg.SetAdmin("", "")
				// сохраняем конфигурацию
				if err := cfg.Save(); err != nil {
					return err
				}
				return c.Send(nil)
			},
		},
	}, rest.BasicAuth(cfg.AuthorizeAdmin, "Pusher Admin")) // требуется доступ администратора

	// работа с клиентами для уведомлений, пользователями и токенами
	mux.Handles(rest.Paths{
		"/apns": {
			// возвращает список зарегистрированных тем с обработчиками
			"GET": func(c *rest.Context) error {
				return c.Send(rest.JSON{"topics": cfg.Topics()})
			},
		},
		"/apns/:id": {
			// возвращает информацию о сертификате
			"GET": func(c *rest.Context) error {
				client := cfg.Client(c.Param("id"))
				if client == nil {
					return c.Send(rest.ErrNotFound)
				}
				return c.Send(client.CertificateInfo)
			},
		},
		"/apns/:id/push": {
			// отправляет уведомление на все устройства указанных пользователей
			"POST": func(c *rest.Context) error {
				id := c.Param("id")
				client := cfg.Client(id)
				if client == nil {
					return c.Send(rest.ErrNotFound)
				}
				var n struct {
					Users       []string    // список пользователей
					Payload     interface{} // содержимое сообщения
					Expiration  time.Time   // время жизни
					LowPriority bool        // низкий приоритет
				}
				// разбираем заголовок с типом информации в запросе
				mediatype, _, _ := mime.ParseMediaType(
					c.Request.Header.Get("Content-Type"))
				// в зависимости от формата запроса, разбираем данные
				if mediatype == "application/json" {
					if err := c.Bind(&n); err != nil {
						return c.Send(err)
					}
				} else {
					n.LowPriority = (c.PostFormValue("lowPriority") != "")
					if exp := c.PostFormValue("expiration"); exp != "" {
						var err error
						n.Expiration, err = time.Parse(time.RFC3339, exp)
						if err != nil {
							return c.Error(http.StatusBadRequest, err.Error())
						}
					}
					n.Users = c.PostForm["user"]
					if mediatype == "application/x-www-form-urlencoded" {
						n.Payload = c.PostFormValue("payload")
					} else if mediatype == "multipart/form-data" {
						file, _, err := c.FormFile("payload")
						if err != nil {
							return c.Error(http.StatusBadRequest, err.Error())
						}
						data, err := ioutil.ReadAll(file)
						file.Close()
						if err != nil {
							return c.Error(http.StatusBadRequest, err.Error())
						}
						n.Payload = data
					} else {
						return c.Send(rest.ErrUnsupportedMediaType)
					}
				}
				if len(n.Users) == 0 {
					var err error
					n.Users, err = cfg.store.GetUsers(id)
					if err != nil {
						return err
					}
				}
				var result = make(map[string]int, len(n.Users))
				for _, user := range n.Users {
					tokens, err := cfg.store.GetUserTokens(id, user)
					if err != nil {
						return err
					}
					result[user] = len(tokens)
					err = client.Push(Notification{
						User: user,
						Notification: apns.Notification{
							Topic:       id,
							Expiration:  n.Expiration,
							LowPriority: n.LowPriority,
							Payload:     n.Payload,
						},
					}, tokens...)
					if err != nil {
						return c.Error(http.StatusBadRequest, err.Error())
					}
				}
				return c.Send(rest.JSON{"push": result})
			},
		},
		"/apns/:id/users": {
			// возвращает список пользователей, зарегистрированных за данным
			// обработчиком уведомлений
			"GET": func(c *rest.Context) error {
				id := c.Param("id")
				client := cfg.Client(id)
				if client == nil {
					return c.Send(rest.ErrNotFound)
				}
				users, err := cfg.store.GetUsers(id)
				if err != nil {
					return err
				}
				return c.Send(rest.JSON{"users": users})
			},
		},
		"/apns/:id/users/:name": {
			// возвращает список токенов, зарегистрированных за данным
			// пользователем для данного типа уведомлений
			"GET": func(c *rest.Context) error {
				id := c.Param("id")
				client := cfg.Client(id)
				if client == nil {
					return c.Send(rest.ErrNotFound)
				}
				tokens, err := cfg.store.GetUserTokens(id, c.Param("name"))
				if err != nil {
					return err
				}
				return c.Send(rest.JSON{"tokens": tokens})
			},
			// регистрирует новый токен для данного пользователя
			"POST": func(c *rest.Context) error {
				id := c.Param("id")
				client := cfg.Client(id)
				if client == nil {
					return c.Send(rest.ErrNotFound)
				}
				var token struct{ Token string }
				if err := c.Bind(&token); err == rest.ErrUnsupportedMediaType {
					token.Token = c.PostFormValue("token")
				} else if err != nil {
					return c.Send(err)
				}
				if l := len(token.Token); l < 64 || l > 200 {
					return c.Error(http.StatusBadRequest, "Bad token size")
				}
				if _, err := hex.DecodeString(token.Token); err != nil {
					return c.Error(http.StatusBadRequest, err.Error())
				}
				created, err := cfg.store.AddUserToken(id, c.Param("name"),
					token.Token)
				if err != nil {
					return err
				}
				// сохраняем конфигурацию
				if err := cfg.Save(); err != nil {
					return err
				}
				if created {
					c.Status(http.StatusCreated)
				}
				return c.Send(nil)
			},
			// удаляет все токены данного пользователя для указанного
			// клиента уведомлений
			"DELETE": func(c *rest.Context) error {
				id := c.Param("id")
				client := cfg.Client(id)
				if client == nil {
					return c.Send(rest.ErrNotFound)
				}
				exists, err := cfg.store.DeleteUser(id, c.Param("name"))
				if err != nil {
					return err
				}
				if exists {
					// сохраняем конфигурацию
					if err := cfg.Save(); err != nil {
						return err
					}
				} else {
					return c.Send(rest.ErrNotFound)
				}
				return c.Send(nil)
			},
		},
		"/apns/:id/users/:name/push": {
			// отправляет уведомление на все устройства пользователя
			"POST": func(c *rest.Context) error {
				id := c.Param("id")
				client := cfg.Client(id)
				if client == nil {
					return c.Send(rest.ErrNotFound)
				}
				var n struct {
					Payload     interface{} // содержимое сообщения
					Expiration  time.Time   // время жизни
					LowPriority bool        // низкий приоритет
				}
				// разбираем заголовок с типом информации в запросе
				mediatype, _, _ := mime.ParseMediaType(
					c.Request.Header.Get("Content-Type"))
				// в зависимости от формата запроса, разбираем данные
				if mediatype == "application/json" {
					if err := c.Bind(&n); err != nil {
						return c.Send(err)
					}
				} else {
					n.LowPriority = (c.PostFormValue("lowPriority") != "")
					if exp := c.PostFormValue("expiration"); exp != "" {
						var err error
						n.Expiration, err = time.Parse(time.RFC3339, exp)
						if err != nil {
							return c.Error(http.StatusBadRequest, err.Error())
						}
					}
					if mediatype == "application/x-www-form-urlencoded" {
						n.Payload = c.PostFormValue("payload")
					} else if mediatype == "multipart/form-data" {
						file, _, err := c.FormFile("payload")
						if err != nil {
							return c.Error(http.StatusBadRequest, err.Error())
						}
						data, err := ioutil.ReadAll(file)
						file.Close()
						if err != nil {
							return c.Error(http.StatusBadRequest, err.Error())
						}
						n.Payload = data
					} else {
						return c.Send(rest.ErrUnsupportedMediaType)
					}
				}
				user := c.Param("name")
				tokens, err := cfg.store.GetUserTokens(id, user)
				if err != nil {
					return err
				}
				var sent = len(tokens)
				err = client.Push(Notification{
					User: user,
					Notification: apns.Notification{
						Topic:       id,
						Expiration:  n.Expiration,
						LowPriority: n.LowPriority,
						Payload:     n.Payload,
					},
				}, tokens...)
				if err != nil {
					return c.Error(http.StatusBadRequest, err.Error())
				}
				return c.Send(rest.JSON{"push": sent})
			},
		},
	}, rest.BasicAuth(cfg.Authorize, "Pusher")) // авторизация пользователя
}
