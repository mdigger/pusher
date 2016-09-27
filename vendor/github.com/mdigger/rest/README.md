# Web API

[![GoDoc](https://godoc.org/github.com/mdigger/rest?status.svg)](https://godoc.org/github.com/mdigger/rest)
[![Build Status](https://travis-ci.org/mdigger/rest.svg)](https://travis-ci.org/mdigger/rest)
[![Coverage Status](https://coveralls.io/repos/mdigger/rest/badge.svg?branch=master&service=github)](https://coveralls.io/github/mdigger/rest?branch=master)

Библиотека для быстрого описания Web API.

Вообще, библиотека написана исключительно для внутреннего использования и
нет никаких гарантий, что она не будет время от времени серьезно изменяться.
Поэтому, если вы хотите использовать ее в своих проектах, то делайте fork.


## Достоинства

Основные достоинства в том, что она компактная и минималистская, облегчает
некоторые часто используемые вещи, поддерживает параметры в пути, совместима
со стандартной библиотекой [http](https://golang.org/pkg/net/http/) 
и позволяет описывать обработчики в таком виде, как удобно мне.

Батарейки входят в комплект: я не стал разделять библиотеку на модули, а
решил включить поддержку тех вещей, которые мне обычно требуются. В
частности, сжатие ответов, вывод логов доступа и защита от ошибок в
обработчиках поддерживаются сразу и по умолчанию включены.

Библиотека поддерживает простую отдачу данных в формате `JSON`. Если
необходимо поддерживать другие форматы (например, **MsgPack**), можно
заменить Encoder на поддерживающий нужный формат. 

С поддержкой _middleware_ тоже сложилось все достаточно красиво: можно
указывать список обработчиков, которые будут выполняться последовательно.
А чтобы не пришлось это делать для каждого запроса отдельно, можно
воспользоваться вспомогательными функциями, позволяющими определять сразу
много путей и методов одновременно. Для передачи значений между
обработчиками в библиотеке предусмотрено внутреннее хранилище, которое
живет только в контексте запроса и автоматически освобождается при окончании
обработки.


## Поддержка параметров в путях

В общем, это было одним из основных моментов, который побудил меня написать
данную библиотеку: очень не хватало возможности в стандартной библиотеке
задать задавать параметры и легко их получать в обработчике. Не скажу, что
решение, лежащее в основе данной реализации, является самым правильным,
быстрым, компактным или оптимальным. Но оно работает с приемлемой для меня
скоростью и пости не требует никаких ресурсов.

Синтаксис, используемый для задания параметров пути, достаточно традиционен
и встречается почти во всех аналогичных библиотеках: символ `:` в начала
задает именованный параметр пути, а символ `*` может использоваться в
качестве завершающего параметра, который "сожрет" весь оставший путь. Вот
несколько примеров задания пути:

	/users/:id
	/users/:id/files
	/users/:id/files/*file

Количество элементов пути ограничено 32768 элементами, но я сильно надеюсь,
что этого мне хватит для любых проектов. Еще одно ограничение: параметр
`*` можно использовать только в самом конце пути и после него не должно
быть никаких других элементов.

Для чтения значения параметра пути используется функция контекста:

	id := c.Param("id")


## Контекст

Да, я пошел порочным путем и объединил `http.Request` с `http.ResponseWriter`
в одном своем объекте `Context`, добавив к нему некоторые вспомогательные
функции. С одной стороны, это не очень правильно. Но, черт возьми, красиво
и просто **СУЩЕСТВЕННО** сокращает количество символов, которые нужно набрать,
чтобы описать обработчик запроса. А еще это позволило обеспечить прозрачную
поддержку некоторым удобным вещам.


## Пример

```go
package main

import (
	"net/http"
	"os"

	"github.com/mdigger/rest"
)

func main() {
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
```