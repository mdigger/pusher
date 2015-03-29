package pusher

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/alexjlockwood/gcm"

	"github.com/mdigger/apns"
)

// Config описывает настройки сервера.
type Config struct {
	DB     string                        `json:"db"`     // имя файла с хранилищем
	Server string                        `json:"server"` // адрес и порт для запуска сервиса
	Apps   map[string]map[string]*Bundle `json:"apps"`   // список поддерживаемых приложений с разбиением по bundleId
}

// Bundle описывает информацию для подключения к сервису.
type Bundle struct {
	// тип соединения: должно быть "apns"
	Type string `json:"type"`
	// идентификатор приложения
	BundleID string `json:"bundleId"`
	// флаг соединения с отладочным сервером
	Sandbox bool `json:"sandbox,omitempty"`
	// сертификаты TLS
	Certificate [][]byte `json:"certificate"`
	// приватный ключ
	PrivateKey []byte `json:"privateKey"`
	// ключ для отсылки GCM
	ApiKey string `json:"apiKey"`
	// клиент для отсылки уведомлений
	apnsClient *apns.Client
	// конфигурация для подключения к APNS
	apnsConfig *apns.Config
	// клиент для отправки GCM
	gcmClient *gcm.Sender
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
	// инициализируем клиентов для всех приложений
	for _, bundles := range config.Apps {
		for _, bundle := range bundles {
			switch bundle.Type {
			case "apns":
				cert, err := tls.X509KeyPair(
					bytes.Join(bundle.Certificate, []byte{'\n'}), bundle.PrivateKey)
				if err != nil {
					return nil, err
				}
				var conf = &apns.Config{
					BundleID:    bundle.BundleID,
					Sandbox:     bundle.Sandbox,
					Certificate: cert,
				}
				conf.SetLogger(nil)
				bundle.apnsConfig = conf
				bundle.apnsClient = apns.NewClient(conf)
			case "gcm":
				bundle.gcmClient = &gcm.Sender{
					ApiKey: bundle.ApiKey,
					Http:   http.DefaultClient,
				}
			}
		}
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
