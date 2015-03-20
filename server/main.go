package main

import (
	"github.com/mdigger/pusher"
	"log"
	"net/http"
)

func main() {
	config, err := pusher.LoadConfig("pusher.json") // Читаем конфигурационный файл
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
	log.Fatal(http.ListenAndServe(config.Server, mux)) // стартуем сервис HTTP
}