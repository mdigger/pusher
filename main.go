package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/mdigger/log"
)

var (
	appName = "pusher"                      // название сервиса
	version = "2.1.29"                      // версия
	date    = "2016-10-04"                  // дата сборки
	build   = ""                            // номер сборки в git-репозитории
	host    = "pushsvr.connector73.net:443" // адрес сервера и порт
	config  = appName + ".json"             // имя конфигурационного файла
	agent   = fmt.Sprintf("%s/%s", appName, version)
)

func main() {
	log.SetLevel(log.DebugLevel)
	log.SetFlags(0)
	// выводим информацию о версии сборки
	log.WithFields(log.Fields{
		"version": version,
		"date":    date,
		"build":   build,
		"name":    appName,
	}).Info("starting service")

	// разбираем параметры запуска приложения
	flag.StringVar(&config, "config", config, "config `filename`")
	flag.StringVar(&host, "address", host, "server address and `port`")
	flag.Parse()

	// загружаем конфигурацию сервиса
	log.WithField("file", config).Info("loading config")
	serviceConfig, err := LoadConfig(config)
	if err != nil {
		log.WithError(err).Error("loading config error")
		os.Exit(1)
	}
	defer serviceConfig.Close()
	// инициализируем сервис
	var service = NewService(serviceConfig)
	// инициализируем HTTP-сервер
	server := &http.Server{
		Addr:         host,
		Handler:      service.mux,
		ReadTimeout:  time.Second * 60,
		WriteTimeout: time.Second * 120,
	}
	// для защищенного соединения проделываем дополнительные настройки
	host, port, err := net.SplitHostPort(host)
	if err != nil {
		log.WithError(err).Error("bad server address")
		os.Exit(2)
	}
	if port == "https" || port == "443" {
		if host != "localhost" && host != "127.0.0.1" {
			manager := autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(host),
				Email:      "dmitrys@xyzrd.com",
				Cache:      autocert.DirCache("letsEncript.cache"),
			}
			server.TLSConfig = &tls.Config{
				GetCertificate: manager.GetCertificate,
			}
		} else {
			// исключительно для отладки
			cert, err := tls.X509KeyPair(LocalhostCert, LocalhostKey)
			if err != nil {
				panic(fmt.Sprintf("local certificates error: %v", err))
			}
			server.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
		}
		// запускаем автоматический переход для HTTP на HTTPS
		go func() {
			log.Info("starting http redirect")
			err := http.ListenAndServe(":http", http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r,
						"https://"+r.Host+r.URL.String(),
						http.StatusMovedPermanently)
				}))
			if err != nil {
				log.WithError(err).Warning("http redirect server error")
			}
		}()
		// запускаем основной сервер
		go func() {
			log.WithFields(log.Fields{
				"address": server.Addr,
				"host":    host,
			}).Info("starting https")
			err = server.ListenAndServeTLS("", "")
			// корректно закрываем сервисы по окончании работы
			log.WithError(err).Warning("https server stoped")
			os.Exit(3)
		}()
	} else {
		// не защищенный HTTP сервер
		go func() {
			log.WithField("address", server.Addr).Info("starting http")
			err = server.ListenAndServe()
			log.WithError(err).Warning("http server stoped")
			os.Exit(3)
		}()
	}

	// инициализируем поддержку системных сигналов и ждем, когда он случится
	monitorSignals(os.Interrupt, os.Kill)
	log.Info("service stoped")
}

// monitorSignals запускает мониторинг сигналов и возвращает значение, когда
// получает сигнал. В качестве параметров передается список сигналов, которые
// нужно отслеживать.
func monitorSignals(signals ...os.Signal) os.Signal {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, signals...)
	return <-signalChan
}
