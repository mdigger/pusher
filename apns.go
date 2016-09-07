package main

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/mdigger/apns3"
)

// PoolCount описывает количество создаваемых клиентов в пуле.
var PoolCount uint = 4

// apnsCertificate добавляет к сертификату флаг sandbox.
type apnsCertificate struct {
	tls.Certificate      // сертификат
	Sandbox         bool // флаг использования development версии
}

// APNS описывает список зарегистрированных сертификатов и инициализированных
// клиентов для отправки уведомлений Apple Push.
type APNS struct {
	// список разобранных сертификатов, сгруппированных по bundle ID
	certificates map[string]*apnsCertificate
	// список клиентов для отправки уведомлений, сгруппированный по
	// поддерживаемым темам и bundle ID
	clients map[string]*Client
	// канал с результатами отправки уведомлений
	responses chan Response
	// функция для удаления токенов из хранилища
	deleteUserToken func(id, user, token string, timestamp time.Time) error
	// блокировка одновременного изменения данных
	mu sync.RWMutex
}

// ErrBadCertificate описывает ошибку невалидного сертификата, используемую
// во всяких непонятных ситуациях с сертификатом.
var ErrBadCertificate = errors.New("bad certificate")

// AddCertificate добавляет новый сертификат в список поддерживаемых сертификатов.
// Одновременно создаются и клиенты для отсылки уведомлений для всех
// поддерживаемых тем сертификата, а старые удаляются. Возвращает true, если
// клиенты были заменены и false, если раньше клиенты с таким идентификатором
// не были зарегистрированы.
func (c *APNS) AddCertificate(certificate tls.Certificate, sandbox bool) (
	info *apns.CertificateInfo, replaced bool, err error) {
	client := apns.New(certificate)
	if client.CertificateInfo == nil {
		return nil, false, ErrBadCertificate
	}
	client.Sandbox = sandbox
	c.mu.Lock()
	if c.responses == nil {
		c.responses = make(chan Response)
		go c.apnsResponses()
	}
	pool := NewClient(client, PoolCount, c.responses)
	if c.clients == nil {
		c.clients = make(map[string]*Client)
	}
	// сначала удаляем все зарегистрированных клиентов
	if oldClient, ok := c.clients[client.BundleID]; ok {
		replaced = true
		oldClient.Close() // закрываем пул соединений
		if topics := oldClient.Topics; len(topics) > 0 {
			for _, topic := range topics {
				delete(c.clients, topic)
			}
		} else {
			delete(c.clients, oldClient.BundleID)
		}
	}
	if c.certificates == nil {
		c.certificates = make(map[string]*apnsCertificate)
	}
	c.certificates[client.BundleID] = &apnsCertificate{certificate, sandbox}
	// добавляем в список клиента, ассоциируя его со всеми темами, которые
	// он поддерживает
	if topics := client.Topics; len(topics) > 0 {
		for _, topic := range topics {
			c.clients[topic] = pool
		}
	} else {
		c.clients[client.BundleID] = pool
	}
	c.mu.Unlock()
	return client.CertificateInfo, replaced, nil
}

// Remove удаляет сертификат с указанным идентификатором и всех связанных
// с ним клиентов. Возвращает true, если клиенты с таким идентификатором были
// зарегистрированы.
func (c *APNS) Remove(id string) (exists bool) {
	c.mu.Lock()
	if c.clients == nil || c.certificates == nil {
		c.mu.Unlock()
		return false
	}
	if client, ok := c.clients[id]; ok {
		exists = true
		client.Close() // закрываем пул соединений
		if topics := client.Topics; len(topics) > 0 {
			for _, topic := range topics {
				delete(c.clients, topic)
			}
		} else {
			delete(c.clients, id)
		}
	}
	delete(c.certificates, id)
	c.mu.Unlock()
	return exists
}

// SetSandbox устанавливает флаг Sandbox для указанного сертификата.
func (c *APNS) SetSandbox(id string, sandbox bool) bool {
	c.mu.Lock()
	if cert, ok := c.certificates[id]; ok {
		cert.Sandbox = sandbox
	} else {
		c.mu.Unlock()
		return false
	}
	if client, ok := c.clients[id]; ok {
		client.Sandbox = sandbox
	}
	c.mu.Unlock()
	return true
}

// Client возвращает инициализированного клиента для отправки уведомлений
// для указанного идентификатора. Если клиент для обработки указанного
// идентификатора не зарегистрирован, то возвращается nil.
func (c *APNS) Client(id string) *Client {
	c.mu.RLock()
	client := c.clients[id]
	c.mu.RUnlock()
	return client
}

// Certificates возвращает список bundle ID всех зарегистрированных
// сертификатов и связанный с ними флаг Sandbox.
func (c *APNS) Certificates() []string {
	c.mu.RLock()
	list := make([]string, 0, len(c.certificates))
	for name := range c.certificates {
		list = append(list, name)
	}
	c.mu.RUnlock()
	sort.Strings(list)
	return list
}

// Topics возвращает список всех поддерживаемых тем для всех сертификатов.
func (c *APNS) Topics() []string {
	c.mu.RLock()
	list := make([]string, 0, len(c.clients))
	for name := range c.clients {
		list = append(list, name)
	}
	c.mu.RUnlock()
	sort.Strings(list)
	return list
}

// certificateData представляет сертификат в сохраняемом виде.
type certificateData struct {
	Certificate [][]byte `json:"cert"`
	PrivateKey  []byte   `json:"private"`
	Sandbox     bool     `json:"sandbox,omitempty"`
}

// getCertificates возвращает список с данными сертификатов для сохранения.
func (c *APNS) getCertificates() []*certificateData {
	c.mu.RLock()
	list := make([]*certificateData, 0, len(c.certificates))
	for _, cert := range c.certificates {
		list = append(list, &certificateData{
			Certificate: cert.Certificate.Certificate,
			PrivateKey: x509.MarshalPKCS1PrivateKey(
				cert.Certificate.PrivateKey.(*rsa.PrivateKey)),
			Sandbox: cert.Sandbox,
		})
	}
	c.mu.RUnlock()
	return list
}

// restoreConfig воссоздает новое описание конфигурации из списка сертификатов.
func restoreCertificates(list []*certificateData) (*APNS, error) {
	var config = new(APNS)
	for _, c := range list {
		if len(c.Certificate) < 1 {
			return nil, ErrBadCertificate
		}
		leaf, err := x509.ParseCertificate(c.Certificate[0])
		if err != nil {
			return nil, err
		}
		privateKey, err := x509.ParsePKCS1PrivateKey(c.PrivateKey)
		if err != nil {
			return nil, err
		}
		_, _, err = config.AddCertificate(tls.Certificate{
			Certificate: c.Certificate,
			PrivateKey:  privateKey,
			Leaf:        leaf,
		}, c.Sandbox)
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}
