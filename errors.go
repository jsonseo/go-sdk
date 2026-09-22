package jsonseo

import (
	"errors"
	"fmt"
	"sort"
)

// Сигнальные ошибки для errors.Is. Отказы сервиса приезжают как *APIError,
// который сам сопоставляется с нужной из них по HTTP-статусу.
var (
	// ErrUnauthorized — 403 или 401: ключ не передан или недействителен.
	ErrUnauthorized = errors.New("jsonseo: ключ не передан или недействителен")
	// ErrPaymentRequired — 402: на счёте не хватает средств.
	ErrPaymentRequired = errors.New("jsonseo: на счёте не хватает средств")
	// ErrValidation — 422: параметры не приняты. Деньги не списываются.
	ErrValidation = errors.New("jsonseo: параметры запроса не приняты")
	// ErrRateLimited — 429: превышен лимит частоты.
	ErrRateLimited = errors.New("jsonseo: превышен лимит частоты")
	// ErrServiceUnavailable — 503: выдачу получить не вышло, деньги не списаны.
	ErrServiceUnavailable = errors.New("jsonseo: выдачу получить не вышло")
	// ErrTimeout — ответа не дождались. Автоматически не повторяется.
	ErrTimeout = errors.New("jsonseo: ответа не дождались")
	// ErrIncompleteResponse — пришло меньше, чем обещал сервис.
	ErrIncompleteResponse = errors.New("jsonseo: ответ пришёл не целиком")
	// ErrNetwork — до сервиса не достучались: сеть, DNS, TLS.
	ErrNetwork = errors.New("jsonseo: запрос не удался")
	// ErrParse — ответ пришёл, но не разобрался как JSON.
	ErrParse = errors.New("jsonseo: ответ не разобрался как JSON")
	// ErrInvalidArgument — SDK забраковал аргументы, запрос не отправлялся.
	ErrInvalidArgument = errors.New("jsonseo: неверный аргумент")
)

// APIError — сервис ответил отказом. Тело сохраняется целиком.
type APIError struct {
	// Status — HTTP-статус ответа.
	Status int
	// Message — сообщение сервиса.
	Message string
	// Body — тело ответа как есть.
	Body string
	// Payload — разобранный JSON ответа. Пустой, если тело не разобралось.
	Payload map[string]any
	// Errors — ошибки по именам параметров, приходят с 422.
	Errors map[string][]string
	// RetryAfter — через сколько секунд вернуться. 0, если срок не назван.
	RetryAfter int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("jsonseo: %s (HTTP %d)", e.Message, e.Status)
}

// Is сопоставляет отказ с сигнальной ошибкой по статусу, чтобы работал
// errors.Is(err, jsonseo.ErrPaymentRequired).
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrUnauthorized:
		return e.Status == 401 || e.Status == 403
	case ErrPaymentRequired:
		return e.Status == 402
	case ErrValidation:
		return e.Status == 422
	case ErrRateLimited:
		return e.Status == 429
	case ErrServiceUnavailable:
		return e.Status == 503
	}

	return false
}

// Fields — забракованные параметры, если отказ пришёл с 422. Порядок
// обхода map случаен, поэтому имена сортируются: иначе лог и сравнение
// плясали бы от запуска к запуску.
func (e *APIError) Fields() []string {
	fields := make([]string, 0, len(e.Errors))

	for field := range e.Errors {
		fields = append(fields, field)
	}

	sort.Strings(fields)

	return fields
}

// TransportError — обмен не состоялся или оборвался. Kind говорит, какой
// именно: от него зависит, повторяем мы запрос или нет.
type TransportError struct {
	Kind error
	Err  error
}

func (e *TransportError) Error() string {
	if e.Err == nil {
		return e.Kind.Error()
	}

	return fmt.Sprintf("%s: %s", e.Kind, e.Err)
}

func (e *TransportError) Unwrap() error { return e.Err }

func (e *TransportError) Is(target error) bool { return target == e.Kind }

// ParseError — успех, но тело не разобралось как JSON. Тело сохраняется:
// страница уже оплачена, и достать из неё данные руками лучше, чем ничего.
type ParseError struct {
	Body string
	Err  error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s: %s", ErrParse, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

func (e *ParseError) Is(target error) bool { return target == ErrParse }

func newTransportError(kind error, err error) error {
	return &TransportError{Kind: kind, Err: err}
}

// apiErrorFor собирает отказ под этот HTTP-статус.
func apiErrorFor(status int, body string, payload map[string]any, retryAfter int) *APIError {
	message, _ := payload["message"].(string)

	if message == "" {
		message = fmt.Sprintf("JSON SEO API вернул ошибку %d", status)
	}

	return &APIError{
		Status:     status,
		Message:    message,
		Body:       body,
		Payload:    payload,
		Errors:     fieldErrors(payload),
		RetryAfter: retryAfter,
	}
}

func fieldErrors(payload map[string]any) map[string][]string {
	raw, ok := payload["errors"].(map[string]any)
	if !ok {
		return nil
	}

	errs := make(map[string][]string, len(raw))

	for field, messages := range raw {
		switch value := messages.(type) {
		case []any:
			for _, message := range value {
				errs[field] = append(errs[field], fmt.Sprint(message))
			}
		default:
			errs[field] = []string{fmt.Sprint(value)}
		}
	}

	return errs
}
