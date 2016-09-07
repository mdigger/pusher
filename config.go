package main

import (
	"encoding/gob"
	"os"
)

// Config описывает конфигурацию сервиса.
type Config struct {
	Authorization        // авторизация
	APNS                 // список сертификатов и клиентов
	store         *Store // хранилище пользователей и токенов устройств
	filename      string // имя файла с конфигурацией
}

// Save сохраняет конфигурацию в файл.
func (c *Config) Save() error {
	filename := c.filename
	if filename == "" {
		filename = "config.gob"
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	err = gob.NewEncoder(file).Encode(configData{
		Admin:        c.administrator,
		Users:        c.getUsers(),
		Certificates: c.getCertificates(),
	})
	file.Close()
	return err
}

// LoadConfig загружает и разбирает сохраненную конфигурацию из файла.
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	var data configData
	err = gob.NewDecoder(file).Decode(&data)
	file.Close()
	if err != nil {
		return nil, err
	}
	apns, err := restoreCertificates(data.Certificates)
	if err != nil {
		return nil, err
	}
	return &Config{
		Authorization: Authorization{
			administrator: data.Admin,
			users:         restoreUsers(data.Users),
		},
		APNS:     *apns,
		filename: filename,
	}, nil
}

// configData информация о конфигурации для сохранения.
type configData struct {
	Admin        *user              // административная учетная запись
	Users        []user             // список пользователей
	Certificates []*certificateData // список сертификатов
}
