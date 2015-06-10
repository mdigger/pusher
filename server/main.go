package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mdigger/pusher"
)

func main() {
	var filename = os.Args[0]
	if ext := filepath.Ext(filename); ext != "" {
		filename = filename[:len(filename)-len(ext)]
	}
	config, err := pusher.LoadConfig(filename + ".json") // Читаем конфигурационный файл
	if err != nil {
		log.Fatalln("Error loading config:", err)
	}
	mux := http.NewServeMux()                              // Инициализируем обработчики HTTP-запросов
	httpservice, err := pusher.NewHTTPService(config, mux) // Формируем обработчик запросов
	if err != nil {
		log.Fatalln("Error creating service:", err)
	}
	defer httpservice.Close() // закрываем по окончании
	log.Println("Running", config.Server)
	var currentDir = filepath.Dir(os.Args[0]) // текущий каталог
	// стартуем сервис HTTP
	log.Fatal(http.ListenAndServeTLS(config.Server,
		filepath.Join(currentDir, "cert.pem"), filepath.Join(currentDir, "key.pem"), mux))
}
