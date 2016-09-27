package rest

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/mdigger/log"
	"github.com/mdigger/router"
)

// Context содержит контекстную информацию HTTP-запроса и методы формирования
// ответа на них. Т.к. http.Request импортируется в Context напрямую, то можно
// использовать все его свойства и методы, как родные свойства и методы самого
// контекста.
//
// Context скрывает http.ResponseWriter от прямого использования и, вместо
// этого, предоставляет свои методы для формирования ответа. Это позволяет
// обойти некоторые скользкие моменты и, иногда, несколько упростить код.
//
// Однако и без некоторой ложки дегтя не обошлось: функция Context.Header()
// скрывает доступ заголовкам запроса. Поэтому приходится явно прописывать,
// что необходимо обращение именно к ним.
type Context struct {
	*http.Request        // HTTP запрос
	ContentType   string // тип информации в ответе

	response http.ResponseWriter // ответ на запрос
	params   router.Params       // именованные параметры из пути запроса
	status   int                 // код HTTP-ответа
	sended   bool                // флаг отосланного ответа
	query    url.Values          // параметры запроса в URL (кеш)
	size     int                 // размер переданных данных
	writer   io.Writer           // интерфейс для записи ответов
	compress bool                // флаг, что мы включили сжатие
	log      *log.TraceContext   // для вывода лога
}

// GetHeader позволяет получить доступ к заголовкам http,Request, которые
// оказались скрытыми из-за объединения запроса и ответа в одном объекте.
// Так что это просто короткий путь доступа к ним, чтобы не писать что-то типа
// c.Request.Header.Get("Context-Type").
func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

// Header возвращает HTTP-заголовки ответа. Используется для поддержки
// интерфейса http.ResponseWriter.
func (c *Context) Header() http.Header {
	return c.response.Header()
}

// WriteHeader записывает заголовок ответа. Вызов метода автоматически взводит
// внутренний флаг, что отправка ответа начата. После его вызова отсылка
// каких-либо данных другим способом, кроме Write, уже не поддерживается.
// Используется для поддержки интерфейса http.ResponseWriter.
func (c *Context) WriteHeader(code int) {
	// log.WithField("code", code).Debug("WriteHeader")
	if c.sended {
		return
	}
	c.status = code
	if c.status == 0 {
		c.status = http.StatusOK
	} else if c.status < 100 || c.status >= 600 {
		c.status = http.StatusInternalServerError
	}
	c.sended = true
	c.response.WriteHeader(c.status)
}

// Write записывает данные в качестве ответа сервера. Может вызываться несколько
// раз. Используется для поддержки интерфейса http.ResponseWriter.
//
// При первом вызове (может быть не явный) автоматически устанавливается статус
// ответа. Если статус ответа не был задан, то будет использован статус 200
// (ОК). Так же, если не был задан ContentType, то он будет определен
// автоматически на основании анализа первых байт данных.
func (c *Context) Write(data []byte) (int, error) {
	// log.WithField("length", len(data)).Debug("Write")
	if !c.sended {
		// выполняем только при первой отдаче данных
		header := c.response.Header()
		if header.Get("Content-Type") == "" {
			if c.ContentType == "" {
				// если тип не установлен, то анализируем его на основании
				// содержимого ответа
				c.ContentType = http.DetectContentType(data)
			}
			header.Set("Content-Type", c.ContentType)
		}
		// перед первой отдачей данных отдаем статус ответа
		c.WriteHeader(c.status)
	}
	// записываем данные в качестве ответа
	n, err := c.writer.Write(data)
	c.size += n
	return n, err
}

// Flush отдает накопленный буфер с ответом. Используется для поддержки
// интерфейса http.Flusher.
func (c *Context) Flush() {
	// log.Debug("Flush")
	c.response.(http.Flusher).Flush()
	if gzw, ok := c.writer.(*gzip.Writer); ok {
		gzw.Flush()
	}
}

// Hijack используется для перехвата управления над ответами сервера. Например,
// для поддержки Websocket.
func (c *Context) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return c.response.(http.Hijacker).Hijack()
}

// CloseNotify поддерживает интерфейс http.CloseNotifier.
func (c *Context) CloseNotify() <-chan bool {
	return c.response.(http.CloseNotifier).CloseNotify()
}

// Status устанавливает код HTTP-ответа, который будет отправлен сервером. Вызов
// данного метода не приводит к немедленной отправке ответа, а только
// устанавливает внутренний статус. Статус должен быть в диапазоне от 200 до
// 599, в противном случае статус не изменяется.
//
// Метод возвращает ссылку на основной контекст, чтобы можно было использовать
// его в последовательности выполнения команд. Например, можно сразу установить
// код ответа и тут же опубликовать данные.
func (c *Context) Status(code int) *Context {
	if !c.sended && code >= 200 && code < 600 {
		c.status = code
	}
	return c
}

// Param возвращает значение именованного параметра. Если параметр с таким
// именем не найден, то возвращается значение параметра из URL с тем же именем.
//
// Разобранные параметры запроса пути сохраняются внутри Context и повторного
// его разбора уже не требует. Но это происходит только при первом к ним
// обращении.
func (c *Context) Param(key string) string {
	for _, param := range c.params {
		if param.Key == key {
			return param.Value
		}
	}
	if c.query == nil {
		c.query = c.Request.URL.Query()
	}
	return c.query.Get(key)
}

// Data возвращает пользовательские данные, сохраненные в контексте запроса с
// указанным ключем. Обычно эти данные используются, когда необходимо передать
// их между несколькими обработчиками.
func (c *Context) Data(key interface{}) interface{} {
	// возвращаем данные из контекста запроса
	return c.Request.Context().Value(key)
}

// SetData сохраняет пользовательские данные в контексте запроса с указанным
// ключем.
//
// Рекомендуется в качестве ключа использовать какой-нибудь приватный тип и его
// значение, чтобы избежать случайного затирания данных другими обработчиками:
// это гарантированно обезопасит от случайного доступа к ним. Но строки тоже
// поддерживаются. :)
func (c *Context) SetData(key, value interface{}) {
	// инициализируем новый Context, добавив в него наш ключ и значение
	ctx := context.WithValue(c.Request.Context(), key, value)
	// подменяем запрос на новый, с сохраненным контекстом
	c.Request = c.Request.WithContext(ctx)
}

// Bind разбирает данные запроса и заполняет ими указанный в параметре объект.
// Разбор осуществляется с помощью Encoder.
func (c *Context) Bind(obj interface{}) error {
	return Encoder.Bind(c, obj)
}

// Send отсылает переданные данные как ответ на запрос. В зависимости от типа
// данных, используются разные форматы ответов. Поддерживаются данные в формате
// string, error, []byte, io.Reader и nil. Все остальные типы данных приводятся
// к формату JSON.
//
// Данный метод можно использовать только один раз: после того, как ответ
// отправлен, повторный вызов данного метода сразу возвращает ошибку.
func (c *Context) Send(data interface{}) (err error) {
	// log.WithField("data", data).Debug("Send")
	// не можем отправить ответ, если он уже отправлен
	// вместо этого используйте метод Write
	if c.sended {
		return ErrDataAlreadySent
	}
	// в зависимости от типа данных, отдаем их разными способами
	switch data := data.(type) {
	case nil:
		// удаляем заголовки сжатия, если они были установлены
		if c.compress {
			header := c.Header()
			header.Del("Content-Encoding")
			header.Del("Vary")
			// сбрасываем сжатие и возвращаем стандартный ResponseWriter
			if gzw, ok := c.writer.(*gzip.Writer); ok {
				gzw.Reset(ioutil.Discard)
				gzw.Close()
				gzips.Put(gzw)
				c.writer = c.response
			}
		}
		// отдаем статус
		if c.status == 0 {
			c.status = http.StatusNoContent
		}
		c.WriteHeader(c.status)
	case string:
		if c.ContentType == "" {
			c.ContentType = "text/plain; charset=utf-8"
		}
		_, err = io.WriteString(c, data)
	case error:
		err = c.sendError(data)
	case []byte:
		_, err = c.Write(data)
	case io.Reader:
		_, err = io.Copy(c, data)
	default: // кодируем как объект
		err = Encoder.Encode(c, data)
	}
	return err
}

// Error отправляет указанный текст как описание ошибки. В зависимости от
// флага EncodeError, данный текст будет отдан как описание или как JSON с кодом
// статуса. В отличии от обычных ошибок, на данный текст не распространяется
// правило отладки и текст будет отдан в неизменном виде, в не зависимости от
// установленного значения Debug.
func (c *Context) Error(code int, msg string) error {
	return c.sendError(&HTTPError{code, msg})
}

// Redirect отсылает ответ с требованием временного перехода по указанному URL.
func (c *Context) Redirect(urlStr string, code int) error {
	if u, err := url.Parse(urlStr); err == nil {
		if u.Scheme == "" && u.Host == "" {
			oldpath := c.Request.URL.Path
			if oldpath == "" { // should not happen, but avoid a crash if it does
				oldpath = "/"
			}
			// no leading http://server
			if urlStr == "" || urlStr[0] != '/' {
				// make relative path absolute
				olddir, _ := path.Split(oldpath)
				urlStr = olddir + urlStr
			}
			var query string
			if i := strings.Index(urlStr, "?"); i != -1 {
				urlStr, query = urlStr[:i], urlStr[i:]
			}
			// clean up but preserve trailing slash
			trailing := strings.HasSuffix(urlStr, "/")
			urlStr = path.Clean(urlStr)
			if trailing && !strings.HasSuffix(urlStr, "/") {
				urlStr += "/"
			}
			urlStr += query
		}
	}
	c.Header().Set("Location", urlStr)
	if code < 300 || code >= 400 {
		code = http.StatusFound
	}
	c.Status(code)
	if EncodeError {
		return c.Send(JSON{
			"code":     code,
			"message":  http.StatusText(code),
			"location": urlStr,
		})
	}
	if c.Request.Method == http.MethodGet {
		c.ContentType = "text/html; charset=utf-8"
		return c.Send(fmt.Sprintf("<a href=\"%s\">%s</a>\n",
			html.EscapeString(urlStr), http.StatusText(code)))
	}
	return nil
}

// ServeContent просто вызывает http.ServeContent, передавая ему все
// необходимые параметры. Т.к. стандартная функция не подразумевает возврата
// какой либо ошибки, то и здесь ошибку вы не получите.
func (c *Context) ServeContent(name string, modtime time.Time,
	content io.ReadSeeker) error {
	http.ServeContent(c, c.Request, name, modtime, content)
	return nil
}

// ServeFile отдает содержимое файла с указанным именем, просто вызывая
// функцию http.ServeFile. Ошибок не возвращает.
func (c *Context) ServeFile(name string) error {
	http.ServeFile(c, c.Request, name)
	return nil
}

// SetCookie добавляет в ответ Cookie.
func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c, cookie)
}

// newContext возвращает новый инициализированный контекст. В отличии от просто
// создания нового контекста, вызов данного метода использует пул контекстов.
func newContext(w http.ResponseWriter, r *http.Request) *Context {
	c := contexts.Get().(*Context)
	// очищаем его от возможных старых данных
	c.Request = r
	c.ContentType = ""
	c.response = w
	c.params = nil
	c.status = 0
	c.sended = false
	c.query = nil
	c.size = 0
	ctxLog := Logger.WithFields(log.Fields{
		"method": r.Method,
		"remote": r.RemoteAddr,
		"url":    r.URL,
	})
	// если сжатие еще не установлено, но поддерживается клиентом, то включаем его
	if Compress && w.Header().Get("Content-Encoding") == "" &&
		strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		gzw := gzips.Get().(*gzip.Writer)
		gzw.Reset(w)
		c.writer = gzw
		c.compress = true
		ctxLog = ctxLog.WithField("gzip", true)
	} else {
		c.writer = w
		c.compress = false
	}
	c.log = ctxLog.Tracef("%s %s", r.Method, r.URL.Path)
	return c
}

// close возвращает контекст в пул используемых контекстов для дальнейшего
// использования. Вызывается автоматически после того, как контекст перестает
// использоваться.
func (c *Context) close(err error) {
	// если ответ не был послан, то шлем ошибку
	if !c.sended {
		c.Send(ErrInternalServerError)
	}
	// если инициализировано сжатие, то закрываем и освобождаем компрессор
	if c.compress {
		if gzw, ok := c.writer.(*gzip.Writer); ok {
			gzw.Flush() // проталкиваем отдачу данных
			gzw.Reset(ioutil.Discard)
			gzw.Close()
			gzips.Put(gzw)
		}
	}
	c.log.WithFields(log.Fields{
		"size":   c.size,
		"status": c.status,
	}).Stop(&err)
	contexts.Put(c) // помещаем контекст обратно в пул
}

// пулы
var (
	contexts = sync.Pool{New: func() interface{} { return new(Context) }}
	gzips    = sync.Pool{New: func() interface{} { return new(gzip.Writer) }}
)
