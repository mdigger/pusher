package service

type APNS struct {
	BundleId string
	Cert     []byte
	Key      []byte
}

type GCM struct {
	BundleId string
	ApiKey   string
}

type Config struct {
	DB       string                            // имя файла с базой данных
	Server   string                            // адрес и порт сервера
	Services map[string]map[string]interface{} // список сервисов по именам / идентификаторам приложений
}
