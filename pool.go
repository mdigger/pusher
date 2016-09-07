package main

import (
	"log"
	"os"
	"time"

	"github.com/mdigger/apns3"
)

// Client описывает пул клиентов для отправки уведомлений,
type Client struct {
	*apns.Client
	notifications chan Notification
	responses     chan<- Response
}

// Response описывает статус отправки уведомлений, возвращаемый в канале.
type Response struct {
	User  string // имя пользователя
	Topic string // тема уведомления
	Token string // токен устройства
	ID    string // идентификатор отсылки уведомления в виде UUID
	Error error  // ошибка отправки уведомления
}

// Notification добавляет к стандартному формату уведомления имя пользователя.
type Notification struct {
	User string
	apns.Notification
}

// NewClient создает пул клиентов для параллельной отправки уведомление.
// workers указывает количество потоков, которое будет инициализировано для
// отправки уведомлений через данного клиента. В responses можно передать канал,
// в который будут отдаваться статусы отправки уведомлений.
func NewClient(c *apns.Client, workers uint, responses chan<- Response) *Client {
	pool := &Client{
		Client:        c,
		notifications: make(chan Notification),
		responses:     responses,
	}
	// startup workers to send notifications
	for i := uint(0); i < workers; i++ {
		go func() {
			for n := range pool.notifications {
				id, err := c.Push(n.Notification)
				if responses != nil {
					responses <- Response{n.User, n.Topic, n.Token, id, err}
				}
			}
		}()
	}
	return pool
}

// Push отправляет уведомления на указанные токены устройств, используя пул
// клиентов. Уведомление на первое устройство всегда отправляется без
// использования пула, чтобы проверить валидность формата уведомления. В случае
// ошибки отправки первого уведомления, не связанной с токеном устройства,
// дальнейшая отправка не происходит и эта ошибка возвращается.
func (p Client) Push(n Notification, tokens ...string) error {
	notification := n.Notification
	// перебираем все токены для отправки, пока они есть
	for len(tokens) > 0 {
		// выбираем из списка токенов самый первый
		notification.Token, tokens = tokens[0], tokens[1:]
		// отправляем уведомление на этот токен
		id, err := p.Client.Push(notification)
		// отправляем статус обработки
		if p.responses != nil {
			p.responses <- Response{n.User, n.Topic, notification.Token, id, err}
		}
		if err == nil {
			break // уведомление успешно отправлено — отправляем остальные
		}
		// если ошибка связана с токеном устройства, то берем следующий токен
		if err, ok := err.(*apns.Error); ok && err.IsToken() {
			continue
		}
		// во всех остальных случаях возвращаем ошибку и прерываем обработку
		return err
	}
	// все остальные токены отправляем в фоне и не ждем окончания их
	// отправки
	go func() {
		for _, token := range tokens {
			n.Notification.Token = token
			p.notifications <- n
		}
	}()
	return nil
}

// Close закрывает канал отсылки уведомлений через пул и прерывает их обработку.
func (p *Client) Close() {
	close(p.notifications)
}

// pushLogger для вывода сообщений о push
var pushLogger = log.New(os.Stdout, "", log.LstdFlags)

// apnsResponses обрабатывает статусы отправки уведомлений.
func (c *APNS) apnsResponses() {
	for r := range c.responses {
		var (
			status      int       // статус отправки уведомления
			msg         string    // сообщение об ошибке
			timestamp   time.Time // временная метка последней валидности токена
			removeToken bool      // флаг удаления токена
		)
		switch err := r.Error.(type) {
		case nil:
			status = 200
		case *apns.Error:
			status = err.Status
			msg = err.Reason
			timestamp = err.Time()
			removeToken = err.IsToken()
		case error:
			status = 500
			msg = err.Error()
		default:
			status = 600
		}
		// выводим лог
		pushLogger.Println(status, r.Topic, r.User, r.Token, r.ID, msg)
		// удаляем токен из хранилища в случае ошибки
		if removeToken && c.deleteUserToken != nil {
			_ = c.deleteUserToken(r.Topic, r.User, r.Token, timestamp)
		}
	}
}
