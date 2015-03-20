package pusher

import (
	"encoding/json"
	"errors"
	"os"
)

// Config описывает настройки сервера.
type Config struct {
	DB     string                        `json:"db"`     // имя файла с хранилищем
	Server string                        `json:"server"` // адрес и порт для запуска сервиса
	Apps   map[string]map[string]*Bundle `json:"apps"`   // список поддерживаемых приложений с разбиением по bundleId
}

// Bundle описывает информацию для подключения к сервису.
type Bundle struct {
	// тип соединения: должно быть "apns" или "gcm"
	Type string `json:"type"`
	// флаг соединения с отладочным сервером (для apns)
	Sandbox bool `json:"sandbox"`
	// сертификаты TLS (для apns)
	Certificate [][]byte `json:"certificate"`
	// приватный ключ (для apns)
	PrivateKey []byte `json:"privateKey"`
	// приватный ключ (для gcm)
	AppKey string `json:"appKey"`
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
		return nil, errors.New("apps not defined")
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
