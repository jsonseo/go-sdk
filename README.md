# JSON SEO Go SDK

Официальный Go-клиент [JSON SEO API](https://jsonseo.ru): выдача Яндекса, Google и Bing, картинки и видео, поисковые подсказки, Яндекс Вордстат, прогноз показов Директа и геолокация по IP.

- Работает на **Go 1.21 и выше**.
- Без зависимостей: только стандартная библиотека.
- Контекст в каждом методе: отмена и дедлайн работают как положено.
- Двадцать один метод сервиса.
- Три попытки на запрос по умолчанию: если сервис затупил, SDK сходит ещё раз сам.

## Установка

```bash
go get github.com/jsonseo/go-sdk
```

Ключ берётся в [личном кабинете](https://jsonseo.ru).

## Быстрый старт

```go
package main

import (
	"context"
	"fmt"
	"log"

	jsonseo "github.com/jsonseo/go-sdk"
)

func main() {
	client, err := jsonseo.New("ВАШ_КЛЮЧ")
	if err != nil {
		log.Fatal(err)
	}

	serp, err := client.Yandex(context.Background(), "купить ноутбук", jsonseo.Params{
		"region": 213,
	})
	if err != nil {
		log.Fatal(err)
	}

	for position, result := range serp.Results {
		fmt.Printf("%d. %s — %s\n", position+1, result.Domain, result.Title)
	}
}
```

Второй аргумент — основной параметр метода, остальное передаётся набором `Params`. Он необязателен:

```go
serp, err := client.Yandex(ctx, "купить ноутбук")
geo, err := client.Geoip(ctx, "77.88.55.242")
balance, err := client.Balance(ctx)
```

---

# Примеры запросов

## Позиции сайта в Яндексе

`break_domain` останавливает сбор на нужном домене — платить за страницы ниже найденной позиции незачем.

```go
serp, err := client.Yandex(ctx, "ремонт айфона", jsonseo.Params{
	"region":       213, // Москва
	"pages":        10,  // до 100 позиций
	"break_domain": "example.com",
})
if err != nil {
	log.Fatal(err)
}

for index, result := range serp.Results {
	if strings.HasSuffix(result.Domain, "example.com") {
		fmt.Println("Позиция:", index+1)

		break
	}
}

fmt.Println("Собрано страниц:", serp.Pages)
fmt.Println("Нашлось всего:", serp.FoundHuman)
```

`serp.Found` — указатель: `nil` означает «разобрать не удалось», а не ноль.

## Выдача Google по нужному городу

Регион задаётся числовым ID из справочника — сервис сам соберёт `uule` и подставит `gl`.

```go
regions, err := client.GoogleRegions(ctx, "Казань")
if err != nil {
	log.Fatal(err)
}

serp, err := client.Google(ctx, "заказать пиццу", jsonseo.Params{
	"region": regions.Regions[0].ID,
	"hl":     "ru",
	"device": "desktop",
	"pages":  2,
})
```

Если Google схлопнул часть результатов как «очень похожие», причина придёт в `serp.FilterDescription`, а вернуть их можно параметром `filter`:

```go
serp, err := client.Google(ctx, "заказать пиццу", jsonseo.Params{"filter": 0})
```

## Выдача Bing

```go
serp, err := client.Bing(ctx, "buy a laptop", jsonseo.Params{
	"mkt":   "en-US",
	"pages": 2,
})

fmt.Println(serp.Mkt, serp.Lang) // фактический рынок и язык
```

## Реклама на странице выдачи

Приходит отдельным срезом, органика не меняется. Стоит +0.01 ₽ за страницу, на которой реклама нашлась.

```go
serp, err := client.Yandex(ctx, "пластиковые окна", jsonseo.Params{
	"region": 213,
	"ads":    true,
})

for _, ad := range serp.Ads {
	fmt.Printf("%s #%d — %s\n", ad.Block, ad.Position, ad.Domain)
	fmt.Println("  ", ad.Title)
}
```

`Block` — где стоял блок: `top` до органики, `bottom` после неё, `inline` между результатами. Пустой непустой срез значит «рекламу просили, но её не было», а `nil` — «не просили».

## Ответ нейросети над выдачей

```go
serp, err := client.Yandex(ctx, "чем отличается osb от фанеры", jsonseo.Params{"ai": true})

if serp.AiAnswer != nil {
	fmt.Println(serp.AiAnswer.Markdown)

	for _, source := range serp.AiAnswer.Sources {
		fmt.Printf("[%d] %s\n", source.ID, source.Domain)
	}
}
```

Стоит +0.01 ₽ и только когда ответ есть: если поисковик его не показал, запрос обойдётся в обычную цену. Доступен только с первой страницы.

## Картинки

```go
images, err := client.YandexImages(ctx, "скандинавский интерьер", jsonseo.Params{
	"orientation": "horizontal",
	"size":        "large",
	"format":      "jpg",
	"pages":       2,
})

for _, image := range images.Results {
	fmt.Printf("%d×%d %s\n", image.Width, image.Height, image.URL)
	fmt.Println("   источник:", image.SourceURL)
}
```

Те же параметры работают у `GoogleImages()` и `BingImages()` — SDK переводит общий фильтр в родной параметр движка. Если у поисковика такого значения нет, придёт `ErrValidation` с указанием, чем заменить.

## Видео

```go
videos, err := client.GoogleVideo(ctx, "как заменить ремень грм", jsonseo.Params{
	"duration": "long",
	"hl":       "ru",
})

for _, video := range videos.Results {
	fmt.Printf("%s — %s\n", video.Title, video.DurationText)
	fmt.Printf("   %s (%s)\n", video.URL, video.Provider)
}
```

`Duration` — указатель на секунды, и `nil` означает «длина неизвестна», а не ноль: у прямых эфиров вместо подписи стоит `LIVE`. Отбор вида `*video.Duration < 600` без проверки на `nil` уронит программу — ориентируйтесь на `DurationText`, он на месте всегда.

## Поисковые подсказки

```go
suggest, err := client.YandexSuggest(ctx, "купить кв", jsonseo.Params{"region": 213})

fmt.Println(suggest.Results)
// [купить квартиру в москве купить квартиру в новостройке ...]
```

Есть у всех трёх поисковиков: `YandexSuggest()`, `GoogleSuggest()`, `BingSuggest()`.

## Справочник регионов

```go
regions, err := client.YandexRegions(ctx, "Казань")

for _, region := range regions.Regions {
	fmt.Printf("%d — %s (%s)\n", region.ID, region.Name, region.Subname)
}
// 43 — Казань (Республика Татарстан)
```

Бесплатно, но ключ нужен: по нему считается лимит запросов в минуту. У `GoogleRegions()` в ответе дополнительно приходит готовая строка `UULE`.

## Вордстат: частота запроса

```go
frequency, err := client.WordstatFrequency(ctx, "ремонт айфона", jsonseo.Params{
	"kind":   "exact", // точная частотность: "!ремонт !айфона"
	"region": 213,
})

fmt.Println(frequency.Results.TotalValue) // 27356
```

Вид частотности задаётся параметром `kind`, кавычки и операторы расставит сервис — фразу передавайте как есть:

| `kind` | Что считает |
| --- | --- |
| `base` | Базовая: фраза как есть |
| `phrase` | Фразовая: `"фраза"` |
| `exact` | Точная: `"!слово !слово"` — для прогноза трафика берут её |
| `superexact` | Сверхточная: `"[!слово !слово]"` |

## Вордстат: расширение семантики

```go
wordstat, err := client.Wordstat(ctx, "ремонт айфона", jsonseo.Params{"region": []int{213, 2}})

for _, phrase := range wordstat.Results.Popular {
	fmt.Println(phrase.Value, phrase.Text)
}

for _, phrase := range wordstat.Results.Associations {
	fmt.Println(phrase.Value, phrase.Text)
}
```

`Popular` — что ищут вместе с фразой, `Associations` — соседняя семантика.

## Вордстат: сезонность

```go
graph, err := client.WordstatGraph(ctx, "купить ёлку", jsonseo.Params{"graph_type": "month"})

for _, point := range graph.Results.Graph {
	fmt.Println(point.Text, point.Absolute)
}
// июнь 2026 9042
// июль 2026 11780
```

`month` и `week` отдают историю с 2018 года, `day` — последние 60 дней.

## Вордстат: география спроса

```go
geo, err := client.WordstatMap(ctx, "купить ноутбук", jsonseo.Params{"map_type": "regions"})

for _, row := range geo.Results.Rows {
	fmt.Println(row.Text, row.Absolute, "индекс", row.Popularity)
}
```

`Popularity` — affinity-индекс: 100 означает средний по стране интерес, выше — повышенный. В каждой строке приходит `RegionID` (указатель, `nil` при неоднозначном названии), его можно сразу подставить в `region` других методов.

## Прогноз показов Яндекс Директа

Рекламный кабинет не нужен. Список фраз передаётся срезом — SDK склеит его сам.

```go
forecast, err := client.Direct(ctx,
	[]string{"ремонт айфона", "замена экрана iphone", `"ремонт айфона"`},
	jsonseo.Params{"region": 213, "period": "month"},
)

for _, row := range forecast.Results {
	fmt.Printf("%s: %d показов\n", row.Phrase, row.Shows)

	for place, bid := range row.Positions {
		fmt.Printf("   %s: ставка %.2f ₽, бюджет %.2f ₽, кликов %d\n",
			place, bid.Bid, bid.Budget, bid.Clicks)
	}
}
```

Вид частотности задаётся операторами прямо во фразе: `ремонт айфона` — базовая, `"ремонт айфона"` — фразовая, `"!ремонт !айфона"` — точная.

Стоимость — 0.01 ₽ за пачку до 4000 символов, это около 150 обычных фраз. За один запрос принимается до 1000 фраз, на аккаунт — не больше 100 запросов в час.

## Геолокация по IP

```go
geo, err := client.Geoip(ctx, "77.88.55.242")

fmt.Println(geo.Country.Name, geo.Region.Name)
fmt.Println(geo.Latitude, geo.Longitude)
```

ID региона тот же, что у Яндекса, — его можно сразу подставить в `region` методов выдачи и Вордстата:

```go
serp, err := client.Yandex(ctx, "доставка пиццы", jsonseo.Params{"region": geo.Region.ID})
```

## Баланс

```go
balance, err := client.Balance(ctx)

fmt.Println(balance.Balance, balance.Currency) // 123.45 RUB
```

---

# Справочник методов

| Метод | Путь API | Что делает |
| --- | --- | --- |
| `Yandex()` | `/yandex` | Органическая выдача Яндекса |
| `YandexSuggest()` | `/yandex/suggest` | Поисковые подсказки |
| `YandexRegions()` | `/yandex/regions` | Справочник регионов, бесплатно |
| `YandexImages()` | `/yandex/images` | Поиск по картинкам |
| `YandexVideo()` | `/yandex/video` | Поиск по видео |
| `Google()` | `/google` | Органическая выдача Google |
| `GoogleSuggest()` | `/google/suggest` | Подсказки |
| `GoogleRegions()` | `/google/regions` | Регионы и готовый `uule`, бесплатно |
| `GoogleImages()` | `/google/images` | Поиск по картинкам |
| `GoogleVideo()` | `/google/video` | Поиск по видео |
| `Bing()` | `/bing` | Органическая выдача Bing |
| `BingSuggest()` | `/bing/suggest` | Подсказки |
| `BingImages()` | `/bing/images` | Поиск по картинкам |
| `BingVideo()` | `/bing/video` | Поиск по видео |
| `Wordstat()` | `/wordstat` | Популярные и похожие запросы |
| `WordstatFrequency()` | `/wordstat/frequency` | Частота запроса одним числом |
| `WordstatGraph()` | `/wordstat/graph` | Динамика по месяцам, неделям, дням |
| `WordstatMap()` | `/wordstat/map` | География показов |
| `Direct()` | `/direct` | Прогноз показов Яндекс Директа |
| `Geoip()` | `/geoip` | Геолокация по IPv4, бесплатно |
| `Balance()` | `/balance` | Остаток на счёте, бесплатно |

Полный список параметров каждого метода — в [документации](https://jsonseo.ru/docs).

Появился метод, которого ещё нет в SDK? Его можно вызвать напрямую:

```go
var result MyType
err := client.Call(ctx, "новый/метод", jsonseo.Params{"параметр": "значение"}, &result)

body, err := client.CallRaw(ctx, "новый/метод", jsonseo.Params{"параметр": "значение"})
```

# Как SDK помогает с параметрами

**Списки передаются срезами.** Фразы для Директа склеиваются переводом строки, остальные списки — запятой:

```go
client.Direct(ctx, []string{"ремонт айфона", "ремонт телефона"})
client.Wordstat(ctx, "ремонт", jsonseo.Params{
	"region": []int{213, 2},
	"device": []string{"desktop", "phone"},
})
```

**Флаги принимаются флагами.** `true` и `false` уезжают как `1` и `0`:

```go
client.Yandex(ctx, "купить ноутбук", jsonseo.Params{"ai": true, "ads": true})
```

**`nil` и пустой срез не отправляются.** Необязательный параметр, который вы ещё не посчитали, можно не вычищать из набора руками.

**Родные параметры поисковиков проходят насквозь.** Вертикали принимают не только общие фильтры, но и `tbs` у Google, `isize` у Яндекса, `qft` у Bing.

# Ошибки

Отказы сервиса приезжают как `*APIError` и сопоставляются с сигнальными ошибками через `errors.Is`.

| Сигнал | Статус | Когда |
| --- | --- | --- |
| `ErrValidation` | 422 | Параметры не приняты. `Errors` — сообщения по полям |
| `ErrUnauthorized` | 403, 401 | Ключ не передан или недействителен |
| `ErrPaymentRequired` | 402 | На счёте не хватает средств |
| `ErrRateLimited` | 429 | Превышен лимит частоты |
| `ErrServiceUnavailable` | 503 | Выдачу получить не вышло. Деньги не списаны |

| Сигнал | Когда |
| --- | --- |
| `ErrNetwork` | До сервиса не достучались: сеть, DNS, TLS |
| `ErrTimeout` | Ответа не дождались |
| `ErrIncompleteResponse` | Соединение оборвалось посреди тела |
| `ErrParse` | Ответ пришёл, но не разобрался как JSON |
| `ErrInvalidArgument` | SDK забраковал аргументы, запрос не отправлялся |

```go
serp, err := client.Yandex(ctx, "купить ноутбук", jsonseo.Params{"pages": 50})

switch {
case errors.Is(err, jsonseo.ErrValidation):
	var apiErr *jsonseo.APIError
	errors.As(err, &apiErr)

	for field, messages := range apiErr.Errors {
		fmt.Println(field, strings.Join(messages, ", "))
	}
case errors.Is(err, jsonseo.ErrPaymentRequired):
	balance, _ := client.Balance(ctx)
	fmt.Println("Баланс кончился:", balance.Balance)
case errors.Is(err, jsonseo.ErrRateLimited):
	var apiErr *jsonseo.APIError
	errors.As(err, &apiErr)

	fmt.Println("Вернуться через", apiErr.RetryAfter, "с")
case err != nil:
	log.Fatal(err)
}
```

У `*APIError` есть `Status`, `Message`, `Body`, разобранный `Payload` и `RetryAfter` — срок, который назвал сервис, если он его назвал.

Отмену и дедлайн вызывающего SDK отдаёт как есть: `errors.Is(err, context.Canceled)` и `errors.Is(err, context.DeadlineExceeded)` работают.

# Повторы

**У каждого запроса три попытки по умолчанию: одна основная и две повторных.** Если сервис затупил и выдачу собрать не вышло (`503`), SDK сам сходит ещё дважды, и обычно этого хватает. `WithAttempts(1)` отключает повторы совсем.

`429`, `5xx` и обрывы связи повторяются автоматически — это ровно те отказы, за которые сервис денег не берёт. Отказы по ключу, балансу и параметрам не повторяются: сами они не изменятся.

Таймаут и оборвавшееся посреди тела соединение не повторяются, и это намеренно: работу на стороне сервиса обрыв у клиента не отменяет — выдача будет собрана и оплачена, а повтор стоил бы ещё раз. Если ответ не успевает прийти, поднимайте `WithTimeout`, а не `WithAttempts`.

Пауза между попытками удваивается и разбавляется случайной добавкой. Если сервис прислал `Retry-After`, SDK не вернётся раньше названного срока. Когда сервис просит ждать дольше `WithMaxRetryDelay`, SDK не ждёт вовсе, а отдаёт ошибку с `RetryAfter` — решение остаётся за вами.

Пауза прерывается контекстом: отмена срабатывает сразу, а не в конце ожидания.

# Настройки клиента

```go
client, err := jsonseo.New("ВАШ_КЛЮЧ",
	jsonseo.WithBaseURL("https://jsonseo.ru/api"),   // адрес API
	jsonseo.WithTimeout(5*time.Minute),              // сколько ждать ответа на попытку
	jsonseo.WithAttempts(3),                         // всего попыток, вместе с первой
	jsonseo.WithRetryDelay(time.Second),             // стартовая пауза между попытками
	jsonseo.WithMaxRetryDelay(30*time.Second),       // потолок паузы
	jsonseo.WithAuthInQuery(),                       // ключ в параметре key вместо заголовка
	jsonseo.WithUserAgent("мой-проект/1.0"),
	jsonseo.WithHTTPClient(myHTTPClient),            // свой http.Client: прокси, транспорт, тесты
)
```

Таймаут по умолчанию намеренно большой: многостраничный запрос выдачи собирается минутами, и обрыв на стороне клиента не отменяет запрос на стороне сервиса — деньги за него уже списаны. Считается он на **каждую попытку** отдельно, а не на весь вызов; общий бюджет задавайте контекстом:

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

serp, err := client.Yandex(ctx, "купить ноутбук")
```

Ключ по умолчанию едет в заголовке `Authorization: Bearer`, а не в адресе: так он не оседает в логах прокси и серверов.

# Разработка

```bash
go test ./...
go vet ./...
```

Тесты идут без внешней сети: поднимается свой `httptest`-сервер на loopback.

# Лицензия

MIT.
