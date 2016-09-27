package rest

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// Эти ошибки обрабатываются при передаче их в метод Context.Send и
// устанавливают соответствующий статус ответа.
//
// Кроме указанных здесь ошибок, так же проверяется, что ошибка отвечает на
// os.IsNotExist (в этом случае статус станет 404) или os.IsPermission (статус
// 403). Все остальные ошибки устанавливают статус 500.
//
// Если вам нет необходимости указывать собственное сообщение для вывода ошибки,
// то проще всего воспользоваться этим предопределенными, использовав их в
// context.Send():
// 	return c.Send(ErrNotfound)
var (
	ErrDataAlreadySent       = &HTTPError{0, "data already sent"}
	ErrBadRequest            = &HTTPError{400, "bad request"}
	ErrUnauthorized          = &HTTPError{401, "unauthorized"}
	ErrForbidden             = &HTTPError{403, "forbidden"}
	ErrNotFound              = &HTTPError{404, "not found"}
	ErrMethodNotAllowed      = &HTTPError{405, "method not allowed"}
	ErrLengthRequired        = &HTTPError{411, "length required"}
	ErrRequestEntityTooLarge = &HTTPError{413, "request entity too large"}
	ErrUnsupportedMediaType  = &HTTPError{415, "unsupported media type"}
	ErrInternalServerError   = &HTTPError{500, "internal server error"}
	ErrNotImplemented        = &HTTPError{501, "not implemented"}
	ErrServiceUnavailable    = &HTTPError{503, "service unavailable"}
)

type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"error"`
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d %s", e.Code, e.Message)
}

// sendError отправляет ответ с ошибкой пользователю.
func (c *Context) sendError(err error) error {
	// приводим формат ошибки к HTTPError
	var httpError *HTTPError
	if herr, ok := err.(*HTTPError); ok {
		if herr.Code == 0 {
			return nil
		}
		httpError = herr
	} else {
		httpError = new(HTTPError)
		switch {
		case os.IsPermission(err):
			httpError.Code = 403
		case os.IsNotExist(err):
			httpError.Code = 404
		default:
			httpError.Code = 500
		}
		if Debug {
			httpError.Message = err.Error()
		} else {
			httpError.Message = http.StatusText(httpError.Code)
		}
	}
	// устанавливаем код ответа
	c.Status(httpError.Code)
	// выводим информацию об ошибке
	if EncodeError {
		return Encoder.Encode(c, httpError)
	}
	c.ContentType = "text/plain; charset=utf-8"
	// это скажет IE, что нет необходимости автоматически определять
	// Content-Type, а необходимо использовать уже отданный content-type.
	c.Header().Set("X-Content-Type-Options", "nosniff")
	_, err = io.WriteString(c, httpError.Message)
	return err
}
