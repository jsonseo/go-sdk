package jsonseo

// SearchResult — органический результат выдачи.
type SearchResult struct {
	URL         string `json:"url"`
	Domain      string `json:"domain"`
	Title       string `json:"title"`
	Passage     string `json:"passage"`
	Breadcrumbs string `json:"breadcrumbs,omitempty"`
	// Mime — тип документа: pdf, doc, xls. Пусто у обычных страниц.
	Mime     string `json:"mime,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// AiAnswerSource — источник, на который опирается ответ нейросети.
type AiAnswerSource struct {
	// ID — номер источника: им подписаны сноски у Яндекса и Bing.
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Domain      string `json:"domain"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	// Citations — сколько сносок ведёт на источник. Только Яндекс и Bing.
	Citations int `json:"citations,omitempty"`
}

// AiAnswer — блок нейросети над выдачей: Алиса AI, AI Overview, Copilot.
type AiAnswer struct {
	Markdown string           `json:"markdown"`
	Sources  []AiAnswerSource `json:"sources,omitempty"`
	// FollowUps — только Яндекс: уточняющие вопросы под ответом.
	FollowUps []string `json:"followUps,omitempty"`
}

// AdSitelink — быстрая ссылка под текстом объявления.
type AdSitelink struct {
	Title       string `json:"title"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

// Ad — рекламное объявление со страницы выдачи.
type Ad struct {
	// Block — top до органики, bottom после неё, inline между результатами.
	Block string `json:"block"`
	// Position — место среди объявлений своего блока, с единицы.
	Position int `json:"position"`
	// Page — страница выдачи: нумерация с нуля и абсолютная.
	Page        int          `json:"page"`
	Domain      string       `json:"domain"`
	URL         string       `json:"url,omitempty"`
	DisplayURL  string       `json:"displayUrl,omitempty"`
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Label       string       `json:"label,omitempty"`
	Format      string       `json:"format,omitempty"`
	Group       int          `json:"group,omitempty"`
	Sitelinks   []AdSitelink `json:"sitelinks,omitempty"`
}

// SearchResponse — ответ органической выдачи, общий для трёх поисковиков.
type SearchResponse struct {
	// Pages — сколько страниц получено: ровно за них и списано.
	Pages int `json:"pages"`
	// Exhausted — выдача закончилась, повторять с большим pages незачем.
	Exhausted bool `json:"exhausted"`
	// BreakDomainHit — поиск остановлен доменом из break_domain.
	BreakDomainHit bool   `json:"breakDomainHit"`
	Query          string `json:"query"`
	RawQuery       string `json:"rawQuery"`
	Region         string `json:"region,omitempty"`
	// FilterDescription — только Google: почему часть результатов скрыта.
	FilterDescription string `json:"filter_description,omitempty"`
	// Found — только Яндекс: сколько нашлось, округлённо самим поисковиком.
	// nil, если разобрать не удалось.
	Found      *int64 `json:"found,omitempty"`
	FoundHuman string `json:"found_human,omitempty"`
	// Mkt и Lang — только Bing: фактические рынок и язык выдачи.
	Mkt  string `json:"mkt,omitempty"`
	Lang string `json:"lang,omitempty"`
	// Lr — только Яндекс: регион, в котором выполнен поиск.
	Lr      int            `json:"lr,omitempty"`
	URL     string         `json:"url,omitempty"`
	Results []SearchResult `json:"results"`
	// AiAnswer приходит только при ai=1 и только если поисковик его показал.
	AiAnswer *AiAnswer `json:"aiAnswer,omitempty"`
	// Ads приходит только при ads=1. Пустой, но не nil срез значит, что
	// рекламу просили, а её не было; nil — что не просили.
	Ads []Ad `json:"ads,omitempty"`
}

// ImageResult — карточка картинки: первые четыре поля есть всегда.
type ImageResult struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Domain    string `json:"domain"`
	SourceURL string `json:"sourceUrl"`
	Thumbnail string `json:"thumbnail,omitempty"`
	// Width и Height — размер оригинала. Ноль означает «поисковик не назвал».
	Width           int `json:"width,omitempty"`
	Height          int `json:"height,omitempty"`
	ThumbnailWidth  int `json:"thumbnailWidth,omitempty"`
	ThumbnailHeight int `json:"thumbnailHeight,omitempty"`
	Bytes           int `json:"bytes,omitempty"`
}

// ImagesResponse — ответ поиска по картинкам.
type ImagesResponse struct {
	Pages     int           `json:"pages"`
	Exhausted bool          `json:"exhausted"`
	Query     string        `json:"query"`
	URL       string        `json:"url,omitempty"`
	Results   []ImageResult `json:"results"`
	Ads       []Ad          `json:"ads,omitempty"`
}

// VideoResult — карточка видео.
type VideoResult struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Domain    string `json:"domain"`
	Thumbnail string `json:"thumbnail,omitempty"`
	// Duration — длительность в секундах. nil означает «длина неизвестна»,
	// а не ноль: у прямых эфиров вместо подписи стоит LIVE.
	Duration      *int   `json:"duration,omitempty"`
	DurationText  string `json:"durationText,omitempty"`
	Published     int64  `json:"published,omitempty"`
	PublishedText string `json:"publishedText,omitempty"`
	// Views — просмотры подписью поисковика: 6,8K. Точного значения нет.
	Views       string `json:"views,omitempty"`
	Provider    string `json:"provider,omitempty"`
	Channel     string `json:"channel,omitempty"`
	Description string `json:"description,omitempty"`
	Breadcrumbs string `json:"breadcrumbs,omitempty"`
}

// VideoResponse — ответ поиска по видео.
type VideoResponse struct {
	Pages     int           `json:"pages"`
	Exhausted bool          `json:"exhausted"`
	Query     string        `json:"query"`
	URL       string        `json:"url,omitempty"`
	Results   []VideoResult `json:"results"`
}

// SuggestResponse — ответ поисковых подсказок.
type SuggestResponse struct {
	Query   string   `json:"query"`
	Results []string `json:"results"`
	// Lr — только Яндекс: регион, в котором собраны подсказки.
	Lr string `json:"lr,omitempty"`
}

// YandexRegion — регион Яндекса из справочника.
type YandexRegion struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Subname string  `json:"subname"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

// YandexRegionsResponse — ответ справочника регионов Яндекса.
type YandexRegionsResponse struct {
	Name    string         `json:"name"`
	Lang    string         `json:"lang"`
	Regions []YandexRegion `json:"regions"`
}

// GoogleRegion — регион Google: вместе с ID приходит готовый uule.
type GoogleRegion struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Subname       string  `json:"subname"`
	Type          string  `json:"type"`
	TypeName      string  `json:"type_name"`
	CanonicalName string  `json:"canonical_name"`
	UULE          string  `json:"uule"`
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
}

// GoogleRegionsResponse — ответ справочника регионов Google.
type GoogleRegionsResponse struct {
	Name    string         `json:"name"`
	Lang    string         `json:"lang"`
	Regions []GoogleRegion `json:"regions"`
}

// WordstatPhrase — строка списка запросов Вордстата.
type WordstatPhrase struct {
	Text  string `json:"text"`
	Value int    `json:"value"`
}

// WordstatResponse — популярные и похожие запросы.
type WordstatResponse struct {
	Text    string `json:"text"`
	Region  string `json:"region"`
	Device  string `json:"device"`
	Results struct {
		// Popular — что ищут вместе с этой фразой.
		Popular []WordstatPhrase `json:"popular"`
		// Associations — соседняя семантика.
		Associations []WordstatPhrase `json:"associations"`
	} `json:"results"`
	// Error — почему данных нет: Вордстат не принял фразу из-за синтаксиса
	// операторов. Ответ при этом удачный и оплаченный, а результаты пустые.
	// У обычного ответа поле пустое.
	Error string `json:"error,omitempty"`
}

// WordstatFrequencyResponse — частота запроса одним числом.
type WordstatFrequencyResponse struct {
	Text    string `json:"text"`
	Region  string `json:"region"`
	Device  string `json:"device"`
	Results struct {
		TotalValue int `json:"totalValue"`
	} `json:"results"`
	// Error — почему данных нет: Вордстат не принял фразу из-за синтаксиса
	// операторов. Ответ при этом удачный и оплаченный, а результаты пустые.
	// У обычного ответа поле пустое.
	Error string `json:"error,omitempty"`
}

// WordstatGraphPoint — точка динамики показов.
type WordstatGraphPoint struct {
	Date string `json:"date"`
	Text string `json:"text"`
	// Absolute — показы за период.
	Absolute int `json:"absolute"`
	// Relative — доля среди всех показов Яндекса.
	Relative float64 `json:"relative"`
}

// WordstatGraphResponse — динамика показов.
type WordstatGraphResponse struct {
	Text    string `json:"text"`
	Region  string `json:"region"`
	Device  string `json:"device"`
	Type    string `json:"type"`
	Results struct {
		Graph []WordstatGraphPoint `json:"graph"`
	} `json:"results"`
	// Error — почему данных нет: Вордстат не принял фразу из-за синтаксиса
	// операторов. Ответ при этом удачный и оплаченный, а результаты пустые.
	// У обычного ответа поле пустое.
	Error string `json:"error,omitempty"`
}

// WordstatMapRow — строка географии показов.
type WordstatMapRow struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Absolute int    `json:"absolute"`
	// Popularity — affinity-индекс: 100 означает средний интерес.
	Popularity float64 `json:"popularity"`
	Relative   float64 `json:"relative"`
	// RegionID годится для параметра region других методов. nil, если
	// название неоднозначно.
	RegionID *int `json:"region_id"`
}

// WordstatMapResponse — география показов.
type WordstatMapResponse struct {
	Text    string `json:"text"`
	Device  string `json:"device"`
	Type    string `json:"type"`
	Results struct {
		Rows []WordstatMapRow `json:"rows"`
	} `json:"results"`
	// Error — почему данных нет: Вордстат не принял фразу из-за синтаксиса
	// операторов. Ответ при этом удачный и оплаченный, а результаты пустые.
	// У обычного ответа поле пустое.
	Error string `json:"error,omitempty"`
}

// DirectPosition — прогноз по одному месту аукциона.
type DirectPosition struct {
	Bid    float64 `json:"bid"`
	Budget float64 `json:"budget"`
	Clicks int     `json:"clicks"`
	CTR    float64 `json:"ctr"`
	Shows  int     `json:"shows"`
}

// DirectForecast — прогноз по одной фразе.
type DirectForecast struct {
	Phrase string `json:"phrase"`
	Shows  int    `json:"shows"`
	// Positions — места аукциона: набор задаёт сам Директ.
	Positions map[string]DirectPosition `json:"positions"`
}

// DirectResponse — прогноз показов Яндекс Директа.
type DirectResponse struct {
	Geo    int    `json:"geo"`
	Period string `json:"period"`
	// Batches — на сколько пачек разбит список: по ним считается стоимость.
	Batches   int              `json:"batches"`
	Processed int              `json:"processed"`
	Results   []DirectForecast `json:"results"`
	Errors    []string         `json:"errors"`
}

// GeoipResponse — геолокация по IP.
type GeoipResponse struct {
	IP        string  `json:"ip"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Region    struct {
		// ID тот же, что у Яндекса: годится для параметра region.
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"region"`
	Country struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		ISOName string `json:"iso_name"`
	} `json:"country"`
}

// BalanceResponse — остаток на счёте.
type BalanceResponse struct {
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}
