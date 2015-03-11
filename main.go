package main

import (
	"github.com/mdigger/pusher/service"
	"log"
	"net/http"
	"strings"
)

func main() {
	config, err := service.LoadConfig("pusher.json") // Читаем конфигурационный файл
	if err != nil {
		log.Fatalln("Error loading config:", err)
	}
	mux := http.NewServeMux()                               // Инициализируем обработчики HTTP-запросов
	httpservice, err := service.NewHTTPService(config, mux) // Формируем обработчик запросов
	if err != nil {
		log.Fatalln("Error creating service:", err)
	}
	defer httpservice.Close() // закрываем по окончании
	log.Println("Supported services:", strings.Join(httpservice.AppIds, ", "))
	log.Println("Running", config.Server)
	log.Fatal(http.ListenAndServe(config.Server, mux)) // стартуем сервис HTTP
}
