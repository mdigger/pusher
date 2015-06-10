package main

import (
	"log"
	"net/http"
	"os"

	"github.com/mdigger/pusher"
)

func main() {
	config, err := pusher.LoadConfig(os.Args[0] + ".json") // Читаем конфигурационный файл
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
	log.Fatal(http.ListenAndServeTLS(config.Server, "cert.pem", "key.pem", mux)) // стартуем сервис HTTP
}
