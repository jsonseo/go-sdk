package jsonseo

import "context"

// --- Яндекс ---

// Yandex — органическая выдача Яндекса: мобильная, регион 213.
// 0.01 ₽ за страницу.
func (c *Client) Yandex(ctx context.Context, text string, params ...Params) (*SearchResponse, error) {
	return fetch[SearchResponse](ctx, c, "yandex", withPrimary("text", text, params))
}

// YandexSuggest — подсказки Яндекса: до 50 фраз с учётом региона.
// 0.01 ₽ за запрос.
func (c *Client) YandexSuggest(ctx context.Context, text string, params ...Params) (*SuggestResponse, error) {
	return fetch[SuggestResponse](ctx, c, "yandex/suggest", withPrimary("text", text, params))
}

// YandexRegions — код региона (lr) по названию города или области.
// Бесплатно, нужен ключ.
func (c *Client) YandexRegions(ctx context.Context, name string, params ...Params) (*YandexRegionsResponse, error) {
	return fetch[YandexRegionsResponse](ctx, c, "yandex/regions", withPrimary("name", name, params))
}

// YandexImages — картинки Яндекса: 20 карточек на страницу.
// 0.01 ₽ за страницу.
func (c *Client) YandexImages(ctx context.Context, query string, params ...Params) (*ImagesResponse, error) {
	return fetch[ImagesResponse](ctx, c, "yandex/images", withPrimary("q", query, params))
}

// YandexVideo — видео Яндекса: 20 карточек на страницу. 0.01 ₽ за страницу.
func (c *Client) YandexVideo(ctx context.Context, query string, params ...Params) (*VideoResponse, error) {
	return fetch[VideoResponse](ctx, c, "yandex/video", withPrimary("q", query, params))
}

// --- Google ---

// Google — органическая выдача google.com: мобильная. 0.01 ₽ за страницу.
func (c *Client) Google(ctx context.Context, query string, params ...Params) (*SearchResponse, error) {
	return fetch[SearchResponse](ctx, c, "google", withPrimary("q", query, params))
}

// GoogleSuggest — подсказки Google: до ~15 фраз. 0.01 ₽ за запрос.
func (c *Client) GoogleSuggest(ctx context.Context, query string, params ...Params) (*SuggestResponse, error) {
	return fetch[SuggestResponse](ctx, c, "google/suggest", withPrimary("q", query, params))
}

// GoogleRegions — ID региона Google по названию и готовый uule.
// Бесплатно, нужен ключ.
func (c *Client) GoogleRegions(ctx context.Context, name string, params ...Params) (*GoogleRegionsResponse, error) {
	return fetch[GoogleRegionsResponse](ctx, c, "google/regions", withPrimary("name", name, params))
}

// GoogleImages — картинки Google: 100 карточек на страницу.
// 0.01 ₽ за страницу.
func (c *Client) GoogleImages(ctx context.Context, query string, params ...Params) (*ImagesResponse, error) {
	return fetch[ImagesResponse](ctx, c, "google/images", withPrimary("q", query, params))
}

// GoogleVideo — видео Google: 10 карточек на страницу. 0.01 ₽ за страницу.
func (c *Client) GoogleVideo(ctx context.Context, query string, params ...Params) (*VideoResponse, error) {
	return fetch[VideoResponse](ctx, c, "google/video", withPrimary("q", query, params))
}

// --- Bing ---

// Bing — органическая выдача bing.com: без локации — Россия.
// 0.01 ₽ за страницу.
func (c *Client) Bing(ctx context.Context, query string, params ...Params) (*SearchResponse, error) {
	return fetch[SearchResponse](ctx, c, "bing", withPrimary("q", query, params))
}

// BingSuggest — подсказки Bing. 0.01 ₽ за запрос.
func (c *Client) BingSuggest(ctx context.Context, query string, params ...Params) (*SuggestResponse, error) {
	return fetch[SuggestResponse](ctx, c, "bing/suggest", withPrimary("q", query, params))
}

// BingImages — картинки Bing: count карточек (по умолчанию 35), дальше
// 700-й Bing не листает.
func (c *Client) BingImages(ctx context.Context, query string, params ...Params) (*ImagesResponse, error) {
	return fetch[ImagesResponse](ctx, c, "bing/images", withPrimary("q", query, params))
}

// BingVideo — видео Bing: count карточек на страницу, по умолчанию 105.
func (c *Client) BingVideo(ctx context.Context, query string, params ...Params) (*VideoResponse, error) {
	return fetch[VideoResponse](ctx, c, "bing/video", withPrimary("q", query, params))
}

// --- Вордстат ---

// Wordstat — популярные и похожие запросы. 0.01 ₽ за запрос.
func (c *Client) Wordstat(ctx context.Context, text string, params ...Params) (*WordstatResponse, error) {
	return fetch[WordstatResponse](ctx, c, "wordstat", withPrimary("text", text, params))
}

// WordstatFrequency — частота запроса одним числом: Results.TotalValue.
// 0.01 ₽ за запрос.
func (c *Client) WordstatFrequency(ctx context.Context, text string, params ...Params) (*WordstatFrequencyResponse, error) {
	return fetch[WordstatFrequencyResponse](ctx, c, "wordstat/frequency", withPrimary("text", text, params))
}

// WordstatGraph — динамика показов: month и week с 2018 года, day —
// последние 60 дней. 0.01 ₽ за запрос.
func (c *Client) WordstatGraph(ctx context.Context, text string, params ...Params) (*WordstatGraphResponse, error) {
	return fetch[WordstatGraphResponse](ctx, c, "wordstat/graph", withPrimary("text", text, params))
}

// WordstatMap — показы по регионам и городам. Popularity — affinity-индекс:
// 100 означает средний интерес. 0.01 ₽ за запрос.
func (c *Client) WordstatMap(ctx context.Context, text string, params ...Params) (*WordstatMapResponse, error) {
	return fetch[WordstatMapResponse](ctx, c, "wordstat/map", withPrimary("text", text, params))
}

// --- Директ и служебные ---

// Direct — прогноз показов Директа со ставками и бюджетом. Кабинет не нужен.
//
// 0.01 ₽ за пачку до 4000 символов (около 150 фраз). До 1000 фраз за запрос,
// 100 запросов в час. Вид частотности задаётся операторами во фразе:
// "ремонт айфона" — фразовая, "!ремонт !айфона" — точная.
func (c *Client) Direct(ctx context.Context, phrases []string, params ...Params) (*DirectResponse, error) {
	return fetch[DirectResponse](ctx, c, "direct", withPrimary("phrases", phrases, params))
}

// Geoip — страна, регион и координаты по IPv4. Бесплатно, нужен ключ.
func (c *Client) Geoip(ctx context.Context, ip string, params ...Params) (*GeoipResponse, error) {
	return fetch[GeoipResponse](ctx, c, "geoip", withPrimary("ip", ip, params))
}

// Balance — текущий баланс. Бесплатно, нужен ключ.
func (c *Client) Balance(ctx context.Context) (*BalanceResponse, error) {
	return fetch[BalanceResponse](ctx, c, "balance", Params{})
}
