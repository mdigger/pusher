package rest_test

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mdigger/rest"
)

func init() {
	rest.Debug = true
}

func Example() {
	var mux = new(rest.ServeMux) // инициализируем обработчик запросов
	// добавляем описание обработчиков, задавая пути, методы и функции их обработки
	mux.Handles(rest.Paths{
		// при задании путей можно использовать именованные параметры с ':'
		"/user/:id": {
			"GET": func(c *rest.Context) error {
				// можно быстро сформировать ответ в JSON
				return c.Send(rest.JSON{"user": c.Param("id")})
			},
			// для одного пути можно сразу задать все обработчики для разных методов
			"POST": func(c *rest.Context) error {
				var data = make(rest.JSON)
				// можно быстро десериализовать JSON, переданный в запросе, в объект
				if err := c.Bind(&data); err != nil {
					// возвращать ошибки тоже удобно
					return err
				}
				return c.Send(rest.JSON{"user": c.Param("id"), "data": data})
			},
		},
		// можно одновременно описать сразу несколько путей в одном месте
		"/message/:text": {
			"GET": func(c *rest.Context) error {
				// параметры пути получаются простым запросом
				return c.Send(rest.JSON{"message": c.Param("text")})
			},
		},
		"/file/:name": {
			"GET": func(c *rest.Context) error {
				// поддерживает отдачу разного типа данных, в том числе и файлов
				file, err := os.Open(c.Param("name") + ".html")
				if err != nil {
					return err
				}
				defer file.Close()
				// можно получать не только именованные элементы пути, но
				// параметры, используемые в запросе
				if c.Param("format") == "raw" {
					c.ContentType = `text/plain; charset="utf-8"`
				} else {
					c.ContentType = `text/html; charset="utf-8"`
				}
				return c.Send(file) // отдаем содержимое файла
			},
		},
		"/favicon.ico": {
			// для работы со статическими файлами определена специальная функция
			"GET": rest.File("./favicon.ico"),
		},
	},
		// добавляем проверку авторизации для всех запросов, определенных выше
		func(c *rest.Context) error {
			// проверяем авторизацию для всех запросов, определенных выше
			login, password, ok := c.BasicAuth()
			if !ok {
				c.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				return c.Send(rest.ErrUnauthorized)
			}
			if login != "login" || password != "password" {
				return c.Send(rest.ErrForbidden)
			}
			return nil
		})
	// можно задать глобальные заголовки для всех ответов
	mux.Headers = map[string]string{
		"X-Powered-By": "My Server",
	}
	// т.к. поддерживается интерфейс http.Handler, то можно использовать
	// с любыми стандартными библиотеками http
	http.ListenAndServe(":8080", mux)
}

var c = new(rest.Context) // test context

func ExampleContext_SetData() {
	type myKeyType byte     // определяем собственный тип данных
	var myKey myKeyType = 1 // генерируем уникальный ключ данных
	// сохраняем данные в контексте, используя уникальный ключ
	c.SetData(myKey, "Test data")
	// читаем данные с помощью ключа
	str := c.Data(myKey).(string)
	fmt.Println(str)
}

func ExampleContext_Send_json() error {
	// отдаем ответ в формате JSON, беря идентификатор пользователя
	// из параметров пути или запроса
	return c.Send(rest.JSON{"user": c.Param("id")})
}

func ExampleContext_Send_file() error {
	// открываем файл
	file, err := os.Open("README.md")
	if err != nil {
		return err
	}
	defer file.Close()
	// устанавливаем тип отдаваемых данных
	c.ContentType = "text/markdown; charset=UTF-8"
	// отдаем содержимое файла в качестве ответа
	return c.Send(file)
}

func ExampleContext_Status() error {
	// возвращаем 201 код окончания
	return c.Status(201).Send(nil)
}

func ExampleContext_Bind() error {
	// инициализируем формат данных для разбора
	data := make(map[string]interface{})
	// читаем запрос и получаем данные в разобранном виде
	if err := c.Bind(&data); err != nil {
		return err
	}
	// возвращаем эти же данные в ответ
	return c.Send(data)
}

var mux = new(rest.ServeMux)

func ExampleHandler_ServeHTTP() {
	http.ListenAndServe(":8080",
		rest.Handler(func(c *rest.Context) error {
			return c.Send(rest.JSON{
				"user": "name",
				"date": time.Now().UTC(),
			})
		}))
}

func ExampleServeMux_Handle() {
	mux.Handle("GET", "/json/",
		func(c *rest.Context) error {
			return c.Send(rest.JSON{
				"user": "name",
				"date": time.Now().UTC(),
			})
		})
}

func ExampleServeMux_ServeHTTP() {
	mux.Handles(rest.Paths{
		"/user/:id": {
			"GET": func(c *rest.Context) error {
				return c.Send(rest.JSON{
					"user": c.Param("id"),
					"date": time.Now().UTC(),
				})
			},
			"POST": rest.Data("OK", "text/plain"),
		},
		"/favicon.ico": {
			"GET": rest.File("./favicon.ico"),
		},
	})
	http.ListenAndServe(":8080", mux)
}

type User struct{}

func (User) get(*rest.Context) error     { return nil }
func (User) post(*rest.Context) error    { return nil }
func secure(h rest.Handler) rest.Handler { return h }

var (
	user       User
	getMessage = func(*rest.Context) error { return nil }
	getFile    = getMessage
)

func ExampleServeMux_Handles() {
	var mux = new(rest.ServeMux)
	mux.Handles(rest.Paths{
		"/user/:id": {
			"GET":  user.get,
			"POST": user.post,
		},
		"/message/:text": {"GET": getMessage},
		"/file/:name":    {"GET": secure(getFile)},
	})
	// т.к. поддерживается интерфейс http.Handler, то можно использовать
	// с любыми стандартными библиотеками
	http.ListenAndServe(":8080", mux)
}

func ExampleRedirect() {
	mux.Handle("GET", "/redirect/", rest.Redirect("/json/"))
}

func ExampleData() {
	mux.Handle("GET", "/static/", rest.Data("OK", ""))
	mux.Handle("GET", "/bin/",
		rest.Data([]byte{0x1, 0x2, 0x3, 0x4}, "application/octet-stream"))
}

func ExampleFile() {
	mux.Handle("GET", "/favicon.ico", rest.File("./favicon.ico"))
}

func ExampleFiles() {
	mux.Handle("GET", "/files/:name", rest.Files("./tmp/"))
}

func ExampleNotImplemented() {
	mux.Handle("GET", "/data/:name", rest.NotImplemented)
}
