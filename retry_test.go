package jsonseo

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestThreeAttemptsByDefault(t *testing.T) {
	// По умолчанию у запроса три попытки: одна основная и две повторных.
	r := newRecorder(t)
	for i := 0; i < 3; i++ {
		r.push(503, `{"message":"Недоступен"}`)
	}
	r.push(200, `{"results":["лишний"]}`)

	// Число попыток не трогаем: проверяется именно значение по умолчанию.
	client, err := New("KEY", WithBaseURL(r.server.URL+"/api"), WithRetryDelay(0), WithMaxRetryDelay(0))
	if err != nil {
		t.Fatalf("клиент не создался: %v", err)
	}

	_, err = client.Yandex(context.Background(), "тест")

	if !errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("ожидался ErrServiceUnavailable, получено %v", err)
	}

	if r.count() != 3 {
		t.Fatalf("ожидалось 3 запроса, отправлено %d", r.count())
	}
}

func TestSlowServiceSucceedsOnALaterAttempt(t *testing.T) {
	// Затупивший сервис успевает ответить с третьей попытки.
	r := newRecorder(t).
		push(503, `{"message":"Недоступен"}`).
		push(503, `{"message":"Недоступен"}`).
		push(200, `{"results":[{"url":"u","domain":"d"}]}`)

	serp, err := r.client(t).Yandex(context.Background(), "тест")
	if err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	if len(serp.Results) != 1 {
		t.Fatalf("ответ разобран неверно: %+v", serp)
	}

	if r.count() != 3 {
		t.Fatalf("ожидалось 3 запроса, отправлено %d", r.count())
	}
}

func TestSingleAttemptMeansNoRetries(t *testing.T) {
	r := newRecorder(t).push(503, `{"message":"Недоступен"}`)

	_, err := r.client(t, WithAttempts(1)).Yandex(context.Background(), "тест")

	if !errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("ожидался ErrServiceUnavailable, получено %v", err)
	}

	if r.count() != 1 {
		t.Fatalf("ожидался 1 запрос, отправлено %d", r.count())
	}
}

func TestServerErrorsAreRetried(t *testing.T) {
	for _, status := range []int{500, 502, 504, 429} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			r := newRecorder(t).push(status, `{"message":"Ошибка"}`).push(200, `{}`)

			if _, err := r.client(t).Yandex(context.Background(), "тест"); err != nil {
				t.Fatalf("запрос не прошёл: %v", err)
			}

			if r.count() != 2 {
				t.Fatalf("статус %d должен повторяться, запросов %d", status, r.count())
			}
		})
	}
}

func TestClientErrorsAreNotRetried(t *testing.T) {
	cases := []struct {
		status int
		signal error
	}{
		{402, ErrPaymentRequired},
		{403, ErrUnauthorized},
		{401, ErrUnauthorized},
		{422, ErrValidation},
	}

	for _, test := range cases {
		t.Run(fmt.Sprint(test.status), func(t *testing.T) {
			r := newRecorder(t).push(test.status, `{"message":"нет"}`)

			_, err := r.client(t).Yandex(context.Background(), "тест")

			if !errors.Is(err, test.signal) {
				t.Fatalf("ожидалась сигнальная ошибка, получено %v", err)
			}

			if r.count() != 1 {
				t.Fatalf("статус %d повторяться не должен, запросов %d", test.status, r.count())
			}
		})
	}
}

func TestValidationCarriesFieldMessages(t *testing.T) {
	r := newRecorder(t).push(422, `{"message":"Введите запрос","errors":{"text":["Введите запрос"]}}`)

	_, err := r.client(t).Yandex(context.Background(), "")

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("ожидался *APIError, получено %v", err)
	}

	if got := apiErr.Errors["text"]; len(got) != 1 || got[0] != "Введите запрос" {
		t.Fatalf("сообщения по полям разобраны неверно: %v", apiErr.Errors)
	}

	if fields := apiErr.Fields(); len(fields) != 1 || fields[0] != "text" {
		t.Fatalf("список полей неверен: %v", fields)
	}
}

func TestRetryAfterIsHonoured(t *testing.T) {
	// Проснуться раньше названного срока — снова получить тот же отказ.
	r := newRecorder(t).
		push(429, `{"message":"Too Many Attempts."}`, map[string]string{"Retry-After": "1"}).
		push(200, `{}`)

	started := time.Now()

	if _, err := r.client(t, WithMaxRetryDelay(30*time.Second)).Yandex(context.Background(), "тест"); err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	// Допуск на зернистость таймера: на Windows сон отмеряется
	// с точностью до миллисекунд в меньшую сторону.
	if elapsed := time.Since(started); elapsed < 950*time.Millisecond {
		t.Fatalf("пауза короче названного срока: %v", elapsed)
	}

	if r.count() != 2 {
		t.Fatalf("ожидалось 2 запроса, отправлено %d", r.count())
	}
}

func TestRetryAfterBeyondCapStopsRetrying(t *testing.T) {
	// Дольше потолка SDK не ждёт: отдаёт ошибку с RetryAfter.
	r := newRecorder(t).push(429, `{"message":"Too Many Attempts."}`, map[string]string{"Retry-After": "600"})

	_, err := r.client(t, WithMaxRetryDelay(30*time.Second)).Yandex(context.Background(), "тест")

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter != 600 {
		t.Fatalf("срок не сохранился: %+v", err)
	}

	if r.count() != 1 {
		t.Fatalf("повторов быть не должно, запросов %d", r.count())
	}
}

func TestRetryAfterAcceptsHTTPDate(t *testing.T) {
	// RFC 9110 разрешает и HTTP-дату.
	when := time.Now().Add(10 * time.Minute).UTC().Format(http.TimeFormat)
	r := newRecorder(t).push(429, `{"message":"Too Many Attempts."}`, map[string]string{"Retry-After": when})

	_, err := r.client(t, WithMaxRetryDelay(30*time.Second)).Yandex(context.Background(), "тест")

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter < 500 {
		t.Fatalf("дата не разобрана: %+v", err)
	}
}

func TestServiceUnavailableKeepsRetryAfter(t *testing.T) {
	r := newRecorder(t).push(503, `{"message":"Недоступен"}`, map[string]string{"Retry-After": "120"})

	_, err := r.client(t, WithMaxRetryDelay(30*time.Second)).Yandex(context.Background(), "тест")

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter != 120 {
		t.Fatalf("срок не сохранился: %+v", err)
	}
}

func TestRepeatedRequestCarriesTheSameBody(t *testing.T) {
	r := newRecorder(t).push(503, `{"message":"Недоступен"}`).push(200, `{}`)

	if _, err := r.client(t).Yandex(context.Background(), "купить ноутбук", Params{"pages": 3}); err != nil {
		t.Fatalf("запрос не прошёл: %v", err)
	}

	if r.took()[0].body != r.took()[1].body {
		t.Fatalf("повтор ушёл с другим телом:\n%s\n%s", r.took()[0].body, r.took()[1].body)
	}
}

func TestTimeoutIsNotRetried(t *testing.T) {
	// Сервис уже считает оплаченный запрос — повтор стоил бы ещё раз.
	var hits atomic.Int64

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer slow.Close()

	client, err := New("KEY",
		WithBaseURL(slow.URL),
		WithTimeout(30*time.Millisecond),
		WithRetryDelay(0),
		WithMaxRetryDelay(0),
	)
	if err != nil {
		t.Fatalf("клиент не создался: %v", err)
	}

	_, err = client.Balance(context.Background())

	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("ожидался ErrTimeout, получено %v", err)
	}

	if got := hits.Load(); got != 1 {
		t.Fatalf("таймаут повторяться не должен, запросов %d", got)
	}
}

func TestIncompleteResponseIsNotRetried(t *testing.T) {
	// Заголовки пришли, а тело оборвалось: выдача уже собрана и оплачена.
	var hits atomic.Int64

	cut := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"results":[`))

		// Рвём соединение на середине обещанного тела.
		if hijacker, ok := w.(http.Hijacker); ok {
			conn, _, err := hijacker.Hijack()
			if err == nil {
				_ = conn.Close()
			}
		}
	}))
	defer cut.Close()

	client, err := New("KEY", WithBaseURL(cut.URL), WithRetryDelay(0), WithMaxRetryDelay(0))
	if err != nil {
		t.Fatalf("клиент не создался: %v", err)
	}

	_, err = client.Balance(context.Background())

	if !errors.Is(err, ErrIncompleteResponse) {
		t.Fatalf("ожидался ErrIncompleteResponse, получено %v", err)
	}

	if got := hits.Load(); got != 1 {
		t.Fatalf("обрыв на отдаче повторяться не должен, запросов %d", got)
	}
}

func TestConnectionRefusedIsRetried(t *testing.T) {
	// Запрос до сервиса не дошёл и ничего не стоил — такой отказ повторяем.
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := dead.URL
	dead.Close()

	client, err := New("KEY", WithBaseURL(url), WithAttempts(2), WithRetryDelay(0), WithMaxRetryDelay(0))
	if err != nil {
		t.Fatalf("клиент не создался: %v", err)
	}

	_, err = client.Balance(context.Background())

	if !errors.Is(err, ErrNetwork) {
		t.Fatalf("ожидался ErrNetwork, получено %v", err)
	}
}

// Недошедший запрос — единственная ветка, где повтор бесплатен. Без этого
// теста её можно выключить целиком, и прогон останется зелёным.
func TestNetworkFailureIsRetriedUntilItSucceeds(t *testing.T) {
	var hits atomic.Int64

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("сокет не открылся: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			// Первую попытку рвём, не ответив: до сервиса запрос не дошёл.
			if hits.Add(1) == 1 {
				_ = conn.Close()

				continue
			}

			go func(conn net.Conn) {
				defer conn.Close()

				buf := make([]byte, 4096)
				_, _ = conn.Read(buf)
				body := `{"balance":1,"currency":"RUB"}`
				_, _ = conn.Write([]byte(
					"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: " +
						strconv.Itoa(len(body)) + "\r\nConnection: close\r\n\r\n" + body,
				))
			}(conn)
		}
	}()

	client, err := New("KEY",
		WithBaseURL("http://"+listener.Addr().String()+"/api"),
		WithRetryDelay(0),
		WithMaxRetryDelay(0),
	)
	if err != nil {
		t.Fatalf("клиент не создался: %v", err)
	}

	balance, err := client.Balance(context.Background())
	if err != nil {
		t.Fatalf("недошедший запрос обязан быть повторён: %v", err)
	}

	if balance.Currency != "RUB" {
		t.Fatalf("ответ разобран неверно: %+v", balance)
	}

	if got := hits.Load(); got != 2 {
		t.Fatalf("ожидалось 2 попытки, сделано %d", got)
	}
}

func TestPauseIsInterruptedByContext(t *testing.T) {
	// Иначе отмена замечалась бы только через всю паузу целиком.
	r := newRecorder(t).
		push(429, `{"message":"Too Many Attempts."}`, map[string]string{"Retry-After": "5"}).
		push(200, `{}`)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	started := time.Now()
	_, err := r.client(t, WithMaxRetryDelay(30*time.Second)).Yandex(ctx, "тест")

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ожидалась отмена, получено %v", err)
	}

	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("отмена ждала конца паузы: %v", elapsed)
	}
}
