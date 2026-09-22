package jsonseo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// recorder — сервер, который запоминает запросы и отдаёт сложенные ответы.
type recorder struct {
	server *httptest.Server
	// Хендлер работает в своей горутине, тест читает из своей.
	mu       sync.Mutex
	requests []recorded
	replies  []reply
}

type recorded struct {
	method  string
	path    string
	headers http.Header
	body    string
	params  url.Values
}

type reply struct {
	status  int
	body    string
	headers map[string]string
}

func newRecorder(t *testing.T) *recorder {
	t.Helper()

	r := &recorder{}
	r.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("тело запроса не прочиталось: %v", err)
		}

		params, _ := url.ParseQuery(string(raw))

		r.mu.Lock()
		r.requests = append(r.requests, recorded{
			method:  req.Method,
			path:    req.URL.Path,
			headers: req.Header.Clone(),
			body:    string(raw),
			params:  params,
		})

		if len(r.replies) == 0 {
			r.mu.Unlock()
			t.Errorf("в очереди не осталось ответов, а запрос пришёл: %s", req.URL.Path)
			w.WriteHeader(http.StatusTeapot)

			return
		}

		next := r.replies[0]
		r.replies = r.replies[1:]
		r.mu.Unlock()

		for name, value := range next.headers {
			w.Header().Set(name, value)
		}

		w.WriteHeader(next.status)
		_, _ = w.Write([]byte(next.body))
	}))

	t.Cleanup(r.server.Close)

	return r
}

func (r *recorder) push(status int, body string, headers ...map[string]string) *recorder {
	next := reply{status: status, body: body}
	if len(headers) > 0 {
		next.headers = headers[0]
	}

	r.mu.Lock()
	r.replies = append(r.replies, next)
	r.mu.Unlock()

	return r
}

// took отдаёт снимок принятых запросов: читать поле напрямую нельзя,
// его пишет горутина хендлера.
func (r *recorder) took() []recorded {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]recorded(nil), r.requests...)
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.requests)
}

func (r *recorder) client(t *testing.T, opts ...Option) *Client {
	t.Helper()

	all := append([]Option{
		WithBaseURL(r.server.URL + "/api"),
		WithRetryDelay(0),
		WithMaxRetryDelay(0),
	}, opts...)

	client, err := New("KEY", all...)
	if err != nil {
		t.Fatalf("клиент не создался: %v", err)
	}

	return client
}

func TestNewRequiresAPIKey(t *testing.T) {
	if _, err := New("   "); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("ожидался ErrInvalidArgument, получено %v", err)
	}
}

func TestNewRejectsMeaninglessAttempts(t *testing.T) {
	for _, attempts := range []int{0, -5} {
		if _, err := New("KEY", WithAttempts(attempts)); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("attempts=%d должен отвергаться, получено %v", attempts, err)
		}
	}
}

func TestSendsKeyInHeaderByDefault(t *testing.T) {
	r := newRecorder(t).push(200, `{"balance":1,"currency":"RUB"}`)

	if _, err := r.client(t).Balance(context.Background()); err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	if got := r.took()[0].headers.Get("Authorization"); got != "Bearer KEY" {
		t.Fatalf("ключ не в заголовке: %q", got)
	}

	if r.took()[0].params.Has("key") {
		t.Fatal("ключ не должен дублироваться в параметрах")
	}
}

func TestSendsKeyInQueryWhenAsked(t *testing.T) {
	r := newRecorder(t).push(200, `{"balance":1,"currency":"RUB"}`)

	if _, err := r.client(t, WithAuthInQuery()).Balance(context.Background()); err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	if r.took()[0].headers.Get("Authorization") != "" {
		t.Fatal("заголовка быть не должно")
	}

	if got := r.took()[0].params.Get("key"); got != "KEY" {
		t.Fatalf("ключ не в параметрах: %q", got)
	}
}

func TestPostsToTheMethodPath(t *testing.T) {
	r := newRecorder(t).push(200, `{"results":[]}`)

	if _, err := r.client(t).Yandex(context.Background(), "купить ноутбук"); err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	if r.took()[0].method != http.MethodPost {
		t.Fatalf("ожидался POST, получен %s", r.took()[0].method)
	}

	if r.took()[0].path != "/api/yandex" {
		t.Fatalf("неверный путь: %s", r.took()[0].path)
	}

	if got := r.took()[0].params.Get("text"); got != "купить ноутбук" {
		t.Fatalf("запрос уехал не в том параметре: %q", got)
	}
}

func TestPrimaryParameterNames(t *testing.T) {
	cases := []struct {
		name  string
		call  func(*Client) error
		param string
		path  string
	}{
		{"yandex", func(c *Client) error { _, err := c.Yandex(context.Background(), "тест"); return err }, "text", "/api/yandex"},
		{"google", func(c *Client) error { _, err := c.Google(context.Background(), "тест"); return err }, "q", "/api/google"},
		{"bing", func(c *Client) error { _, err := c.Bing(context.Background(), "тест"); return err }, "q", "/api/bing"},
		{"regions", func(c *Client) error { _, err := c.YandexRegions(context.Background(), "Казань"); return err }, "name", "/api/yandex/regions"},
		{"geoip", func(c *Client) error { _, err := c.Geoip(context.Background(), "1.2.3.4"); return err }, "ip", "/api/geoip"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			r := newRecorder(t).push(200, `{}`)

			if err := test.call(r.client(t)); err != nil {
				t.Fatalf("запрос не прошёл: %v", err)
			}

			if r.took()[0].path != test.path {
				t.Fatalf("неверный путь: %s", r.took()[0].path)
			}

			if !r.took()[0].params.Has(test.param) {
				t.Fatalf("параметр %s не уехал: %v", test.param, r.took()[0].params)
			}
		})
	}
}

func TestEveryMethodCallsItsOwnPath(t *testing.T) {
	// Опечатка в пути иначе всплыла бы только на боевом ключе.
	ctx := context.Background()

	calls := []struct {
		path string
		call func(*Client) error
	}{
		{"/api/yandex", func(c *Client) error { _, err := c.Yandex(ctx, "т"); return err }},
		{"/api/yandex/suggest", func(c *Client) error { _, err := c.YandexSuggest(ctx, "т"); return err }},
		{"/api/yandex/regions", func(c *Client) error { _, err := c.YandexRegions(ctx, "т"); return err }},
		{"/api/yandex/images", func(c *Client) error { _, err := c.YandexImages(ctx, "т"); return err }},
		{"/api/yandex/video", func(c *Client) error { _, err := c.YandexVideo(ctx, "т"); return err }},
		{"/api/google", func(c *Client) error { _, err := c.Google(ctx, "т"); return err }},
		{"/api/google/suggest", func(c *Client) error { _, err := c.GoogleSuggest(ctx, "т"); return err }},
		{"/api/google/regions", func(c *Client) error { _, err := c.GoogleRegions(ctx, "т"); return err }},
		{"/api/google/images", func(c *Client) error { _, err := c.GoogleImages(ctx, "т"); return err }},
		{"/api/google/video", func(c *Client) error { _, err := c.GoogleVideo(ctx, "т"); return err }},
		{"/api/bing", func(c *Client) error { _, err := c.Bing(ctx, "т"); return err }},
		{"/api/bing/suggest", func(c *Client) error { _, err := c.BingSuggest(ctx, "т"); return err }},
		{"/api/bing/images", func(c *Client) error { _, err := c.BingImages(ctx, "т"); return err }},
		{"/api/bing/video", func(c *Client) error { _, err := c.BingVideo(ctx, "т"); return err }},
		{"/api/wordstat", func(c *Client) error { _, err := c.Wordstat(ctx, "т"); return err }},
		{"/api/wordstat/frequency", func(c *Client) error { _, err := c.WordstatFrequency(ctx, "т"); return err }},
		{"/api/wordstat/graph", func(c *Client) error { _, err := c.WordstatGraph(ctx, "т"); return err }},
		{"/api/wordstat/map", func(c *Client) error { _, err := c.WordstatMap(ctx, "т"); return err }},
		{"/api/direct", func(c *Client) error { _, err := c.Direct(ctx, []string{"т"}); return err }},
		{"/api/geoip", func(c *Client) error { _, err := c.Geoip(ctx, "1.2.3.4"); return err }},
		{"/api/balance", func(c *Client) error { _, err := c.Balance(ctx); return err }},
	}

	r := newRecorder(t)
	for range calls {
		r.push(200, `{}`)
	}

	client := r.client(t)

	for index, test := range calls {
		if err := test.call(client); err != nil {
			t.Fatalf("%s: запрос не прошёл: %v", test.path, err)
		}

		if got := r.took()[index].path; got != test.path {
			t.Fatalf("ожидался путь %s, получен %s", test.path, got)
		}
	}

	if r.count() != len(calls) {
		t.Fatalf("ожидалось %d запросов, отправлено %d", len(calls), r.count())
	}
}

func TestParamsEncoding(t *testing.T) {
	r := newRecorder(t).push(200, `{}`)

	_, err := r.client(t).Wordstat(context.Background(), "ремонт", Params{
		"region":  []int{213, 2},
		"device":  []string{"desktop", "phone"},
		"ai":      true,
		"noreask": false,
		"empty":   []string{},
		"missing": nil,
	})
	if err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	params := r.took()[0].params

	if got := params.Get("region"); got != "213,2" {
		t.Fatalf("список склеен неверно: %q", got)
	}

	if got := params.Get("device"); got != "desktop,phone" {
		t.Fatalf("список склеен неверно: %q", got)
	}

	if params.Get("ai") != "1" || params.Get("noreask") != "0" {
		t.Fatalf("флаги переданы неверно: ai=%q noreask=%q", params.Get("ai"), params.Get("noreask"))
	}

	if params.Has("empty") || params.Has("missing") {
		t.Fatal("пустые значения отправлять не нужно")
	}
}

func TestPhrasesJoinedWithNewlines(t *testing.T) {
	r := newRecorder(t).push(200, `{}`)

	if _, err := r.client(t).Direct(context.Background(), []string{"ремонт айфона", "ремонт телефона"}); err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	if got := r.took()[0].params.Get("phrases"); got != "ремонт айфона\nремонт телефона" {
		t.Fatalf("фразы склеены неверно: %q", got)
	}
}

func TestRejectsUnsupportedParamValue(t *testing.T) {
	r := newRecorder(t)

	_, err := r.client(t).Yandex(context.Background(), "тест", Params{"region": []any{213, struct{}{}}})

	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("ожидался ErrInvalidArgument, получено %v", err)
	}

	if !strings.Contains(err.Error(), "region[1]") {
		t.Fatalf("ошибка не называет место: %v", err)
	}

	if r.count() != 0 {
		t.Fatal("запрос не должен был уйти в сеть")
	}
}

func TestDecodesResponse(t *testing.T) {
	r := newRecorder(t).push(200, `{"pages":2,"found":28000000,"results":[{"url":"https://example.com","domain":"example.com"}]}`)

	serp, err := r.client(t).Yandex(context.Background(), "тест")
	if err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	if serp.Pages != 2 {
		t.Fatalf("pages разобран неверно: %d", serp.Pages)
	}

	if serp.Found == nil || *serp.Found != 28000000 {
		t.Fatalf("found разобран неверно: %v", serp.Found)
	}

	if len(serp.Results) != 1 || serp.Results[0].Domain != "example.com" {
		t.Fatalf("results разобраны неверно: %+v", serp.Results)
	}
}

func TestNullableFieldsStayNil(t *testing.T) {
	// Отсутствие длительности означает «неизвестно», а не ноль: у прямых
	// эфиров вместо подписи стоит LIVE.
	r := newRecorder(t).push(200, `{"results":[{"url":"u","title":"t","domain":"d","durationText":"LIVE"}]}`)

	videos, err := r.client(t).YandexVideo(context.Background(), "тест")
	if err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	if videos.Results[0].Duration != nil {
		t.Fatalf("длительность должна остаться nil, получено %v", *videos.Results[0].Duration)
	}
}

func TestCallRawReturnsBodyAsIs(t *testing.T) {
	body := `<?xml version="1.0"?><yandexsearch></yandexsearch>`
	r := newRecorder(t).push(200, body)

	got, err := r.client(t).CallRaw(context.Background(), "yandex/xml", Params{"query": "тест"})
	if err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	if got != body {
		t.Fatalf("тело изменилось: %q", got)
	}

	if accept := r.took()[0].headers.Get("Accept"); accept != "application/xml, text/xml" {
		t.Fatalf("неверный Accept: %q", accept)
	}
}

func TestUnparsableBodyKeepsIt(t *testing.T) {
	r := newRecorder(t).push(200, `<html>прокси съел ответ</html>`)

	_, err := r.client(t).Balance(context.Background())

	if !errors.Is(err, ErrParse) {
		t.Fatalf("ожидался ErrParse, получено %v", err)
	}

	var parseErr *ParseError
	if !errors.As(err, &parseErr) || parseErr.Body != `<html>прокси съел ответ</html>` {
		t.Fatalf("тело не сохранилось: %+v", err)
	}
}

func TestUserAgent(t *testing.T) {
	r := newRecorder(t).push(200, `{}`).push(200, `{}`)

	if _, err := r.client(t).Balance(context.Background()); err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	if got := r.took()[0].headers.Get("User-Agent"); !strings.HasPrefix(got, "jsonseo-go/") {
		t.Fatalf("неверная подпись: %q", got)
	}

	if _, err := r.client(t, WithUserAgent("мой-проект/1.0")).Balance(context.Background()); err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	if got := r.took()[1].headers.Get("User-Agent"); got != "мой-проект/1.0" {
		t.Fatalf("своя подпись не доехала: %q", got)
	}
}

// Отмена вызывающего обязана остаться отменой, а не стать ErrNetwork:
// иначе SDK счёл бы её повторяемой и пошёл платить за выдачу второй раз.
// errors.Is одного context.Canceled мало — он истинен и через Unwrap.
func TestContextCancellationIsNotWrapped(t *testing.T) {
	r := newRecorder(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := r.client(t).Yandex(ctx, "тест")

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ожидалась отмена вызывающего, получено %v", err)
	}

	var transportErr *TransportError
	if errors.As(err, &transportErr) {
		t.Fatalf("отмена не должна становиться отказом транспорта: %v", transportErr.Kind)
	}

	if errors.Is(err, ErrNetwork) || errors.Is(err, ErrTimeout) {
		t.Fatalf("отмена не должна выглядеть как сбой связи: %v", err)
	}

	if r.count() > 1 {
		t.Fatalf("отменённый запрос не должен повторяться, запросов %d", r.count())
	}
}

// Дедлайн вызывающего — тоже его решение, а не наш таймаут.
func TestCallerDeadlineIsNotOurTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer slow.Close()

	client, err := New("KEY", WithBaseURL(slow.URL), WithTimeout(time.Minute), WithRetryDelay(0))
	if err != nil {
		t.Fatalf("клиент не создался: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err = client.Balance(ctx)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ожидался дедлайн вызывающего, получено %v", err)
	}

	if errors.Is(err, ErrTimeout) {
		t.Fatalf("дедлайн вызывающего не должен выглядеть как наш таймаут: %v", err)
	}
}

func TestPerAttemptTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer slow.Close()

	client, err := New("KEY", WithBaseURL(slow.URL), WithTimeout(30*time.Millisecond), WithAttempts(1))
	if err != nil {
		t.Fatalf("клиент не создался: %v", err)
	}

	_, err = client.Balance(context.Background())

	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("ожидался ErrTimeout, получено %v", err)
	}
}
