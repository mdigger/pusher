package rest

import (
	"encoding/json"
	"mime"
	"strings"
)

// JSON позволяет быстро описать данные в одноименном формате и является
// просто синтаксическим сахаром, ни к чему особо не обязывающим:
// 	return c.Send(rest.JSON{"date": time.Now()})
type JSON map[string]interface{}

// Coder описывает интерфейс для поддержки разбора запроса и кодирования ответа.
// Данная библиотека включает в себя JSONCoder, который поддерживает данный
// интерфейс и позволяет кодировать вывод и декодировать запросы в формате
// JSON. При необходимости вы легко можете заменить его на любой другой,
// поддерживающий данных интерфейс.
type Coder interface {
	Bind(*Context, interface{}) error
	Encode(*Context, interface{}) error
}

// JSONCoder осуществляет разбор запроса и кодирование ответа в формате JSON
// и поддерживает интерфейс Coder. Используется библиотекой по умолчанию
// для вывода информации и разбора запросов в формате JSON. Инициализация
// параметров задается в глобальной переменной Encoder.
type JSONCoder struct {
	MaxBody int64 // максимально допустимый размер запроса
	Indent  bool  // флаг форматированного вывода JSON
}

// Bind разбирает данные запроса в формате JSON и заполняет ими указанный в
// параметре объект.
//
// Если Content-Type запроса не соответствует "application/json", то
// возвращается ошибка ErrUnsupportedMediaType. Так же может возвращать ошибку
// ErrLengthRequired, если не указана длина запроса, ErrRequestEntityTooLarge —
// если запрос превышает значение MaxBody, и ErrBadRequest — если не смогли
// разобрать запрос и поместить результат разбора в объект obj. Все эти ошибки
// поддерживаются методом Send и отдают соответствующий статус ответа на запрос.
func (j JSONCoder) Bind(c *Context, obj interface{}) error {
	r := c.Request // запрос
	// разбираем заголовок с типом информации в запросе
	mediatype, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	charset, ok := params["charset"]
	if !ok {
		charset = "UTF-8"
	}
	// если запрос не является JSON, то возвращаем ошибку
	if mediatype != "application/json" || strings.ToUpper(charset) != "UTF-8" {
		return ErrUnsupportedMediaType
	}
	// если запрос превышает допустимый объем, то возвращаем ошибку
	if j.MaxBody > 0 {
		if r.ContentLength == 0 {
			return ErrLengthRequired
		} else if r.ContentLength > j.MaxBody {
			return ErrRequestEntityTooLarge
		}
	}
	// разбираем данные из запроса
	if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
		return ErrBadRequest
	}
	return nil
}

// Encode кодирует и отправляет ответ с содержимым obj в формате JSON.
func (j JSONCoder) Encode(c *Context, obj interface{}) error {
	if c.ContentType == "" {
		c.ContentType = "application/json; charset=utf-8"
	}
	enc := json.NewEncoder(c)
	if j.Indent {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(obj)
}
