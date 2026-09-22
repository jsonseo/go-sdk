// Package jsonseo — официальный Go SDK для JSON SEO API: выдача Яндекса,
// Google и Bing, картинки и видео, подсказки, Вордстат, прогноз Директа и
// геолокация по IP.
//
//	client, err := jsonseo.New("YOUR_KEY")
//	serp, err := client.Yandex(ctx, "купить ноутбук", jsonseo.Params{"region": 213})
package jsonseo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Version — версия SDK, уезжает в User-Agent.
const Version = "1.0.1"

// DefaultBaseURL — адрес API по умолчанию.
const DefaultBaseURL = "https://jsonseo.ru/api"

// Client — клиент JSON SEO API. Безопасен для одновременного использования.
type Client struct {
	apiKey        string
	baseURL       string
	timeout       time.Duration
	attempts      int
	retryDelay    time.Duration
	maxRetryDelay time.Duration
	authInQuery   bool
	userAgent     string
	httpClient    *http.Client
}

// Option настраивает клиент.
type Option func(*Client) error

// WithBaseURL задаёт адрес API.
func WithBaseURL(url string) Option {
	return func(c *Client) error {
		if url == "" {
			return fmt.Errorf("%w: baseURL не может быть пустым", ErrInvalidArgument)
		}

		c.baseURL = strings.TrimRight(url, "/")

		return nil
	}
}

// WithTimeout задаёт, сколько ждать ответа на одну попытку.
//
// По умолчанию пять минут: многостраничная выдача идёт минутами, и
// оборванный запрос всё равно будет досчитан и оплачен.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) error {
		if timeout <= 0 {
			return fmt.Errorf("%w: timeout должен быть положительным", ErrInvalidArgument)
		}

		c.timeout = timeout

		return nil
	}
}

// WithAttempts задаёт общее число попыток, включая первую. По умолчанию три.
func WithAttempts(attempts int) Option {
	return func(c *Client) error {
		if attempts < 1 {
			return fmt.Errorf("%w: attempts должен быть не меньше 1, получено %d", ErrInvalidArgument, attempts)
		}

		c.attempts = attempts

		return nil
	}
}

// WithRetryDelay задаёт стартовую паузу между попытками.
func WithRetryDelay(delay time.Duration) Option {
	return func(c *Client) error {
		if delay < 0 {
			return fmt.Errorf("%w: retryDelay не может быть отрицательным", ErrInvalidArgument)
		}

		c.retryDelay = delay

		return nil
	}
}

// WithMaxRetryDelay задаёт потолок паузы. Если сервис просит ждать дольше,
// повторов не будет вовсе — в том числе при нуле: он означает «не ждать
// ни секунды», а не «без потолка».
func WithMaxRetryDelay(delay time.Duration) Option {
	return func(c *Client) error {
		if delay < 0 {
			return fmt.Errorf("%w: maxRetryDelay не может быть отрицательным", ErrInvalidArgument)
		}

		c.maxRetryDelay = delay

		return nil
	}
}

// WithAuthInQuery переносит ключ из заголовка в параметр key — там, где
// заголовки до API не доходят.
func WithAuthInQuery() Option {
	return func(c *Client) error {
		c.authInQuery = true

		return nil
	}
}

// WithUserAgent задаёт свою подпись клиента.
func WithUserAgent(agent string) Option {
	return func(c *Client) error {
		c.userAgent = agent

		return nil
	}
}

// WithHTTPClient подставляет свой http.Client — для прокси, своего
// транспорта или тестов.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) error {
		if client == nil {
			return fmt.Errorf("%w: httpClient не может быть nil", ErrInvalidArgument)
		}

		c.httpClient = client

		return nil
	}
}

// New создаёт клиент. Ключ берётся в личном кабинете на jsonseo.ru.
func New(apiKey string, opts ...Option) (*Client, error) {
	// Обрезаем ровно тот же набор, что и остальные SDK: родной trim в
	// каждом языке свой, и один ключ принимался бы по-разному.
	key := strings.Trim(apiKey, " \t\n\r")

	if key == "" {
		return nil, fmt.Errorf("%w: нужен API-ключ, возьмите его на https://jsonseo.ru", ErrInvalidArgument)
	}

	// Заголовок Authorization не переносит не-ASCII и управляющие символы:
	// с таким ключом он не соберётся, и сервис ответит «токен не
	// предоставлен» вместо внятной ошибки.
	for _, r := range key {
		if r < ' ' || r > '~' {
			return nil, fmt.Errorf(
				"%w: API-ключ содержит символы вне ASCII, проверьте, что он скопирован целиком",
				ErrInvalidArgument,
			)
		}
	}

	client := &Client{
		apiKey:        key,
		baseURL:       DefaultBaseURL,
		timeout:       5 * time.Minute,
		attempts:      3,
		retryDelay:    time.Second,
		maxRetryDelay: 30 * time.Second,
		userAgent:     "jsonseo-go/" + Version,
		httpClient:    &http.Client{},
	}

	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// Call — произвольный метод API, если в сервисе появился новый. Ответ
// разбирается в out.
func (c *Client) Call(ctx context.Context, path string, params Params, out any) error {
	body, err := c.do(ctx, path, params, true)
	if err != nil {
		return err
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal([]byte(body), out); err != nil {
		return &ParseError{Body: body, Err: err}
	}

	return nil
}

// CallRaw — то же, но ответ возвращается строкой без разбора.
func (c *Client) CallRaw(ctx context.Context, path string, params Params) (string, error) {
	return c.do(ctx, path, params, false)
}

// fetch выполняет запрос и разбирает ответ в T.
func fetch[T any](ctx context.Context, c *Client, path string, params Params) (*T, error) {
	body, err := c.do(ctx, path, params, true)
	if err != nil {
		return nil, err
	}

	result := new(T)

	if err := json.Unmarshal([]byte(body), result); err != nil {
		return nil, &ParseError{Body: body, Err: err}
	}

	return result, nil
}

// do выполняет запрос, повторяя те отказы, за которые сервис не берёт денег:
// 429, 5xx и обрывы связи до того, как ответ начал приходить.
func (c *Client) do(ctx context.Context, path string, params Params, wantJSON bool) (string, error) {
	values, err := encode(params)
	if err != nil {
		return "", err
	}

	if c.authInQuery {
		values.Set("key", c.apiKey)
	}

	// Всегда POST: длинные списки фраз в GET не помещаются.
	endpoint := c.baseURL + "/" + strings.TrimLeft(path, "/")
	body := values.Encode()

	accept := "application/xml, text/xml"
	if wantJSON {
		accept = "application/json"
	}

	for attempt := 0; ; attempt++ {
		result, err := c.attempt(ctx, endpoint, accept, body)
		if err != nil {
			// Таймаут и обрыв на середине тела не повторяем: выдача уже
			// собрана и оплачена.
			if c.isLastAttempt(attempt) || !errors.Is(err, ErrNetwork) {
				return "", err
			}

			if pauseErr := pause(ctx, c.backoff(attempt)); pauseErr != nil {
				return "", errors.Join(pauseErr, err)
			}

			continue
		}

		if result.status >= 200 && result.status < 300 {
			return result.body, nil
		}

		apiErr := apiErrorFor(result.status, result.body, decodeQuietly(result.body), result.retryAfter)
		wait := time.Duration(result.retryAfter) * time.Second

		// Проснуться раньше названного срока — снова получить тот же отказ.
		// Ждать дольше потолка не станем: отдаём ошибку.
		if c.isLastAttempt(attempt) || !isRetryable(result.status) || wait > c.maxRetryDelay {
			return "", apiErr
		}

		// Не раньше, чем просит сервис, и не чаще своего бэкоффа.
		sleeping := c.backoff(attempt)
		if wait > sleeping {
			sleeping = wait
		}

		if pauseErr := pause(ctx, sleeping); pauseErr != nil {
			return "", errors.Join(pauseErr, apiErr)
		}
	}
}

type attemptResult struct {
	status     int
	body       string
	retryAfter int
}

// attempt — один заход в сеть под собственным таймаутом.
func (c *Client) attempt(parent context.Context, endpoint, accept, body string) (*attemptResult, error) {
	ctx, cancel := context.WithTimeout(parent, c.timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidArgument, err)
	}

	request.Header.Set("Accept", accept)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", c.userAgent)

	if !c.authInQuery {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, c.classify(parent, err, ErrNetwork)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		// Заголовки уже пришли, значит выдача собрана и оплачена: повторять
		// такой обрыв нельзя.
		return nil, c.classify(parent, err, ErrIncompleteResponse)
	}

	return &attemptResult{
		status:     response.StatusCode,
		body:       string(raw),
		retryAfter: parseRetryAfter(response.Header.Get("Retry-After")),
	}, nil
}

// classify отличает отмену снаружи, наш таймаут и всё остальное.
func (c *Client) classify(parent context.Context, err error, fallback error) error {
	// Отмену и дедлайн вызывающего отдаём как есть: это его решение,
	// а не отказ сервиса.
	if parent.Err() != nil {
		return parent.Err()
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return newTransportError(ErrTimeout, err)
	}

	return newTransportError(fallback, err)
}

func (c *Client) isLastAttempt(attempt int) bool {
	// Попытки нумеруются с нуля: при attempts = 3 у последней индекс 2.
	return attempt+1 >= c.attempts
}

// backoff: пауза удваивается с каждой попыткой; случайная добавка разводит
// параллельные запросы, чтобы они не вернулись разом.
func (c *Client) backoff(attempt int) time.Duration {
	delay := float64(c.retryDelay) * math.Pow(2, float64(attempt))
	delay += delay * 0.25 * rand.Float64() //nolint:gosec // джиттер, не криптография

	// Потолок накладывается после добавки, иначе она бы его превышала.
	if time.Duration(delay) > c.maxRetryDelay {
		return c.maxRetryDelay
	}

	return time.Duration(delay)
}

// pause — прерываемая пауза: отмена срабатывает сразу, а не в конце ожидания.
func pause(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isRetryable(status int) bool {
	return status == 429 || status >= 500
}

// decodeQuietly: тело ошибки может быть и не JSON — тогда подробностей нет.
func decodeQuietly(body string) map[string]any {
	var payload map[string]any

	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return map[string]any{}
	}

	if payload == nil {
		return map[string]any{}
	}

	return payload
}

// parseRetryAfter: RFC 9110 разрешает число секунд и HTTP-дату, разбираются
// обе. Ноль означает, что срок не назван.
func parseRetryAfter(header string) int {
	header = strings.TrimSpace(header)

	if header == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(header); err == nil {
		if seconds < 0 {
			return 0
		}

		return seconds
	}

	// http.ParseTime знает все три формата, которые обязывает RFC 9110:
	// RFC 1123, RFC 850 и asctime. mail.ParseDate берёт только первый.
	when, err := http.ParseTime(header)
	if err != nil {
		return 0
	}

	seconds := int(time.Until(when).Seconds())
	if seconds < 0 {
		return 0
	}

	return seconds
}

// withPrimary подставляет основной параметр метода. Пустое значение
// оставляет то, что положили в Params.
func withPrimary(name string, value any, sets []Params) Params {
	params := merge(sets)

	switch typed := value.(type) {
	case string:
		if typed != "" {
			params[name] = typed
		}
	case []string:
		if len(typed) > 0 {
			params[name] = typed
		}
	default:
		if value != nil {
			params[name] = value
		}
	}

	return params
}
