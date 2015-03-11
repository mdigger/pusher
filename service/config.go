package service

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
)

var ErrNoApps = errors.New("no apps defined")

// ApplePushServer описывает конфигурацию для Apple Push Server.
type ApplePushServer struct {
	Gateway  string // адрес сервера
	Feedback string `json:",omitempty"`
	Cert     []byte // публичный ключ
	Key      []byte // приватный ключ
}

// NewApplePushServer возвращает конфигурацию для Apple Push Server, при этом считывая сертификаты,
// необходимые для работы, из файлов. Параметр sandbox указывает, что необходимо использовать
// отладочную версию Apple Push вместо обычной.
func NewApplePushServer(certFile, keyFile string, sandbox bool) (*ApplePushServer, error) {
	certPEMBlock, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	keyPEMBlock, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	var gateway, feedback string
	if sandbox {
		gateway = "gateway.sandbox.push.apple.com:2195"
		feedback = "feedback.sandbox.push.apple.com:2196"
	} else {
		gateway = "gateway.push.apple.com:2195"
		feedback = "feedback.push.apple.com:2196"
	}
	return &ApplePushServer{
		Gateway:  gateway,
		Feedback: feedback,
		Cert:     certPEMBlock,
		Key:      keyPEMBlock,
	}, nil
}

type Application struct {
	Apple *ApplePushServer `json:",omitempty"` // конфигурация Apple Push Server
}

// Config описывает настройки сервера.
type Config struct {
	DB     string                  // имя файла с хранилищем
	Server string                  // адрес и порт для запуска сервиса
	Apps   map[string]*Application // список поддерживаемых приложений
}

// LoadConfig читает конфигурационный файл и возвращает его описание.
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	config := new(Config)
	if err := json.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}
	if len(config.Apps) == 0 {
		return nil, ErrNoApps
	}
	if config.DB == "" {
		config.DB = "pusher.db" // имя файла с базой по умолчанию
	}
	if config.Server == "" {
		config.Server = "localhost:8080" // адрес и порт сервиса по умолчанию
	}
	return config, nil
}

// Save сохраняет конфигурацию в файл.
func (c *Config) Save(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}
