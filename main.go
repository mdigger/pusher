package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/mdigger/rest"
)

var (
	appName = "Pusher"     // название приложения
	version = "2.0.27"     // версия
	date    = "2016-09-07" // дата сборки
	build   = ""           // номер сборки в git-репозитории
)

func main() {
	// выводим версию приложения
	ver := fmt.Sprintf("# %s %s", appName, version)
	if build != "" {
		ver = fmt.Sprintf("%s [git %s]", ver, build)
	}
	ver = fmt.Sprintf("%s (%s)", ver, date)
	fmt.Fprintln(os.Stderr, ver)

	// разбираем параметры запуска приложения
	configFile := flag.String("config", "config.gob", "configuration `file`")
	addr := flag.String("addr", ":8443", "http server address and `port`")
	cert := flag.String("cert", "cert.pem", "server certificate `file`")
	key := flag.String("key", "key.pem", "server private certificate `file`")
	storeDB := flag.String("store", "tokens.db", "db `DSN` connection string")
	compress := flag.Bool("compress", true, "gzip compress response")
	indent := flag.Bool("indent", true, "indent JSON response")
	monitor := flag.Bool("monitor", false, "start monitor handler")
	reset := flag.Bool("reset", false, "remover users and admin authorization")
	flag.UintVar(&PoolCount, "pools", 1, "APNS client pool `size`")
	flag.Parse()

	if PoolCount == 0 {
		PoolCount = 1 // минимальное количество клиентов в пуле
	}
	// загружаем конфигурационный файл
	config, err := LoadConfig(*configFile)
	if err != nil {
		log.Println("Error loading config:", err)
		log.Println("Using empty config")
		// инициализируем пустую конфигурацию
		config = &Config{filename: *configFile}
		if err := config.Save(); err != nil {
			log.Fatalln("Error saving config:", err)
		}
	} else if *reset {
		log.Println("Reset users and admin authorizations")
		config.Reset()
		if err := config.Save(); err != nil {
			log.Fatalln("Error saving config:", err)
		}
	}

	// инициализируем хранилище токенов
	store, err := OpenStore(*storeDB)
	if err != nil {
		log.Fatalln("Error initializing store:", err)
	}
	config.store = store // подключаем хранилище токенов
	// подключаем функцию удаления устаревших и плохих токенов
	config.APNS.deleteUserToken = store.DeleteUserToken

	rest.Compress = *compress // включаем/выключаем сжатие ответов
	// 32 мегабайта и отступы
	rest.Encoder = rest.JSONCoder{1 << 15, *indent}
	rest.Debug = true // включаем вывод информации об ошибках
	// регистрируем обработчики HTTP-запросов
	var mux = new(rest.ServeMux)
	config.registerHandlers(mux) // регистрируем обработчики
	if *monitor {
		registerExpVar(mux) // добавляем монитор
	}
	// запускаем сервис
	go func() {
		log.Printf("Starting service at %q", *addr)
		srv := &http.Server{
			Addr:         *addr,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 120 * time.Second,
		}
		log.Println(srv.ListenAndServeTLS(*cert, *key))
		store.Close() // закрываем соединение с базой
		os.Exit(2)    // останавливаем сервис
	}()

	// go func() {
	// 	for {
	// 		log.Println("Goroutines:", runtime.NumGoroutine())
	// 		time.Sleep(time.Second * 5)
	// 	}
	// }()

	// инициализируем поддержку системных сигналов и ждем, когда он случится...
	monitorSignals(os.Interrupt, os.Kill)
	store.Close() // закрываем соединение с базой
}

// monitorSignals запускает мониторинг сигналов и возвращает значение, когда получает сигнал.
// В качестве параметров передается список сигналов, которые нужно отслеживать.
func monitorSignals(signals ...os.Signal) os.Signal {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, signals...)
	return <-signalChan
}

// registerExpVar регистрирует обработчик, отдающий информацию о запущенном
// процессе.
func registerExpVar(mux *rest.ServeMux) {
	var startTime = time.Now().UTC() // время запуска сервиса
	log.Printf("Monitor handler started at %q", "/debug/vars")
	mux.Handle("GET", "/debug/vars", func(c *rest.Context) error {
		var stats = struct {
			Uptime     int64            `json:"uptime"`
			Goroutines int              `json:"goroutines"`
			NumCPU     int              `json:"numcpu"`
			NumCgoCall int64            `json:"numcgocall"`
			MemStats   runtime.MemStats `json:"memstats"`
		}{
			Uptime:     int64(time.Since(startTime)),
			Goroutines: runtime.NumGoroutine(),
			NumCPU:     runtime.NumCPU(),
			NumCgoCall: runtime.NumCgoCall(),
		}
		runtime.ReadMemStats(&stats.MemStats)
		return c.Send(stats)
	})
}