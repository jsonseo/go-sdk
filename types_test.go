package jsonseo

import (
	"context"
	"testing"
)

// Теги разбора проверяются на живых ответах: опечатка в json-теге не
// ломает сборку и вылезает у пользователя тихим нулём.
func TestResponseShapes(t *testing.T) {
	ctx := context.Background()

	t.Run("подсказки", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"query":"к","results":["купить","куплю"],"lr":"213"}`)

		got, err := r.client(t).YandexSuggest(ctx, "к")
		if err != nil {
			t.Fatal(err)
		}

		if got.Lr != "213" || len(got.Results) != 2 || got.Results[0] != "купить" {
			t.Fatalf("разобрано неверно: %+v", got)
		}
	})

	t.Run("регионы Яндекса", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"name":"Казань","lang":"ru","regions":[
			{"id":43,"name":"Казань","subname":"Республика Татарстан","lat":55.79,"lon":49.1}]}`)

		got, err := r.client(t).YandexRegions(ctx, "Казань")
		if err != nil {
			t.Fatal(err)
		}

		if got.Regions[0].ID != 43 || got.Regions[0].Subname != "Республика Татарстан" {
			t.Fatalf("разобрано неверно: %+v", got.Regions)
		}
	})

	t.Run("регионы Google", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"name":"Казань","lang":"ru","regions":[
			{"id":1012054,"name":"Казань","subname":"Россия","type":"city","type_name":"город",
			 "canonical_name":"Kazan,Russia","uule":"w+CAIQ","lat":55.78,"lon":49.12}]}`)

		got, err := r.client(t).GoogleRegions(ctx, "Казань")
		if err != nil {
			t.Fatal(err)
		}

		region := got.Regions[0]
		if region.TypeName != "город" || region.CanonicalName != "Kazan,Russia" || region.UULE != "w+CAIQ" {
			t.Fatalf("разобрано неверно: %+v", region)
		}
	})

	t.Run("картинки", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"pages":1,"exhausted":false,"query":"кот","results":[
			{"url":"u","title":"t","domain":"d","sourceUrl":"s","thumbnail":"th",
			 "width":800,"height":600,"thumbnailWidth":200,"thumbnailHeight":150,"bytes":1024}]}`)

		got, err := r.client(t).YandexImages(ctx, "кот")
		if err != nil {
			t.Fatal(err)
		}

		image := got.Results[0]
		if image.SourceURL != "s" || image.Width != 800 || image.ThumbnailHeight != 150 || image.Bytes != 1024 {
			t.Fatalf("разобрано неверно: %+v", image)
		}
	})

	t.Run("видео", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"pages":1,"query":"кот","results":[
			{"url":"u","title":"t","domain":"d","duration":1878,"durationText":"31:18",
			 "published":1669000000,"publishedText":"5 мес. назад","views":"6,8K",
			 "provider":"YouTube","channel":"ch","breadcrumbs":"b"}]}`)

		got, err := r.client(t).GoogleVideo(ctx, "кот")
		if err != nil {
			t.Fatal(err)
		}

		video := got.Results[0]
		if video.Duration == nil || *video.Duration != 1878 || video.DurationText != "31:18" {
			t.Fatalf("длительность разобрана неверно: %+v", video)
		}

		if video.PublishedText != "5 мес. назад" || video.Views != "6,8K" || video.Channel != "ch" {
			t.Fatalf("разобрано неверно: %+v", video)
		}
	})

	t.Run("реклама и ответ нейросети", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"pages":1,"query":"окна","results":[],
			"aiAnswer":{"markdown":"текст","sources":[{"id":1,"url":"u","domain":"d","citations":2}],
			            "followUps":["а что если"]},
			"ads":[{"block":"top","position":1,"page":0,"domain":"d","displayUrl":"du",
			        "label":"Реклама","format":"gallery","group":1,
			        "sitelinks":[{"title":"Каталог","url":"u"}]}]}`)

		got, err := r.client(t).Yandex(ctx, "окна")
		if err != nil {
			t.Fatal(err)
		}

		if got.AiAnswer == nil || got.AiAnswer.Sources[0].Citations != 2 || len(got.AiAnswer.FollowUps) != 1 {
			t.Fatalf("ответ нейросети разобран неверно: %+v", got.AiAnswer)
		}

		ad := got.Ads[0]
		if ad.DisplayURL != "du" || ad.Format != "gallery" || ad.Group != 1 || ad.Sitelinks[0].Title != "Каталог" {
			t.Fatalf("реклама разобрана неверно: %+v", ad)
		}
	})

	t.Run("частота Вордстата", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"text":"т","region":"213","device":"desktop","results":{"totalValue":27356}}`)

		got, err := r.client(t).WordstatFrequency(ctx, "т")
		if err != nil {
			t.Fatal(err)
		}

		if got.Results.TotalValue != 27356 {
			t.Fatalf("разобрано неверно: %+v", got)
		}
	})

	t.Run("списки Вордстата", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"text":"т","results":{
			"popular":[{"text":"п","value":1}],"associations":[{"text":"а","value":2}]}}`)

		got, err := r.client(t).Wordstat(ctx, "т")
		if err != nil {
			t.Fatal(err)
		}

		if got.Results.Popular[0].Value != 1 || got.Results.Associations[0].Text != "а" {
			t.Fatalf("разобрано неверно: %+v", got.Results)
		}
	})

	t.Run("динамика Вордстата", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"text":"т","type":"month","results":{"graph":[
			{"date":"2026-06-01","text":"июнь 2026","absolute":9042,"relative":0.0001}]}}`)

		got, err := r.client(t).WordstatGraph(ctx, "т")
		if err != nil {
			t.Fatal(err)
		}

		point := got.Results.Graph[0]
		if point.Date != "2026-06-01" || point.Absolute != 9042 || point.Relative != 0.0001 {
			t.Fatalf("разобрано неверно: %+v", point)
		}
	})

	t.Run("география Вордстата", func(t *testing.T) {
		// region_id приходит и числом, и null: null значит «название неоднозначно».
		r := newRecorder(t).push(200, `{"text":"т","type":"regions","results":{"rows":[
			{"type":"regions","text":"Москва","absolute":111666,"popularity":117.07,"relative":0.0054,"region_id":1},
			{"type":"regions","text":"Центр","absolute":178801,"popularity":108.69,"relative":0.005,"region_id":null}]}}`)

		got, err := r.client(t).WordstatMap(ctx, "т")
		if err != nil {
			t.Fatal(err)
		}

		rows := got.Results.Rows
		if rows[0].RegionID == nil || *rows[0].RegionID != 1 || rows[0].Popularity != 117.07 {
			t.Fatalf("первая строка разобрана неверно: %+v", rows[0])
		}

		if rows[1].RegionID != nil {
			t.Fatalf("неоднозначное название должно давать nil: %+v", rows[1])
		}
	})

	t.Run("прогноз Директа", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"geo":213,"period":"month","batches":1,"processed":1,
			"results":[{"phrase":"ремонт","shows":12345,"positions":{
				"first_place":{"bid":145.5,"budget":4821.3,"clicks":331,"ctr":2.68,"shows":12345}}}],
			"errors":[]}`)

		got, err := r.client(t).Direct(ctx, []string{"ремонт"})
		if err != nil {
			t.Fatal(err)
		}

		place, ok := got.Results[0].Positions["first_place"]
		if !ok || place.Bid != 145.5 || place.CTR != 2.68 || place.Clicks != 331 {
			t.Fatalf("место аукциона разобрано неверно: %+v", got.Results[0].Positions)
		}

		if got.Batches != 1 || got.Geo != 213 {
			t.Fatalf("разобрано неверно: %+v", got)
		}
	})

	t.Run("геолокация", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"ip":"77.88.55.242","latitude":55.75,"longitude":37.62,
			"region":{"id":213,"name":"Москва"},
			"country":{"id":225,"name":"Россия","iso_name":"RU"}}`)

		got, err := r.client(t).Geoip(ctx, "77.88.55.242")
		if err != nil {
			t.Fatal(err)
		}

		if got.Region.ID != 213 || got.Country.ISOName != "RU" || got.Latitude != 55.75 {
			t.Fatalf("разобрано неверно: %+v", got)
		}
	})

	t.Run("баланс", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"balance":123.45,"currency":"RUB"}`)

		got, err := r.client(t).Balance(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if got.Balance != 123.45 || got.Currency != "RUB" {
			t.Fatalf("разобрано неверно: %+v", got)
		}
	})

	t.Run("найдено может быть null", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"pages":1,"query":"т","found":null,"results":[]}`)

		got, err := r.client(t).Yandex(ctx, "т")
		if err != nil {
			t.Fatal(err)
		}

		if got.Found != nil {
			t.Fatalf("null должен давать nil, получено %v", *got.Found)
		}
	})

	t.Run("реклама: пустой список и его отсутствие — разное", func(t *testing.T) {
		r := newRecorder(t).push(200, `{"pages":1,"query":"т","results":[],"ads":[]}`).
			push(200, `{"pages":1,"query":"т","results":[]}`)

		client := r.client(t)

		asked, err := client.Yandex(ctx, "т")
		if err != nil {
			t.Fatal(err)
		}

		if asked.Ads == nil || len(asked.Ads) != 0 {
			t.Fatalf("пустой список должен остаться не-nil: %#v", asked.Ads)
		}

		notAsked, err := client.Yandex(ctx, "т")
		if err != nil {
			t.Fatal(err)
		}

		if notAsked.Ads != nil {
			t.Fatalf("отсутствие поля должно давать nil: %#v", notAsked.Ads)
		}
	})
}

func TestParseRetryAfterEdgeCases(t *testing.T) {
	cases := map[string]int{
		"":                              0,
		"soon":                          0,
		"-5":                            0,
		"0":                             0,
		"17":                            17,
		"  17  ":                        17,
		"Mon, 01 Jan 1990 00:00:00 GMT": 0, // дата в прошлом
	}

	for header, want := range cases {
		if got := parseRetryAfter(header); got != want {
			t.Fatalf("Retry-After %q: ожидалось %d, получено %d", header, want, got)
		}
	}
}

func TestParamsAcceptNamedAndWideTypes(t *testing.T) {
	type device string

	values, err := encode(Params{
		"region":  []int64{213, 2},
		"scores":  []float64{1.5, 2},
		"device":  device("desktop"),
		"flags":   []bool{true, false},
		"pages":   uint8(3),
		"missing": []string{},
	})
	if err != nil {
		t.Fatalf("кодирование не прошло: %v", err)
	}

	if got := values.Get("region"); got != "213,2" {
		t.Fatalf("[]int64 склеен неверно: %q", got)
	}

	if got := values.Get("scores"); got != "1.5,2" {
		t.Fatalf("[]float64 склеен неверно: %q", got)
	}

	if got := values.Get("device"); got != "desktop" {
		t.Fatalf("именованный тип не принят: %q", got)
	}

	if got := values.Get("flags"); got != "1,0" {
		t.Fatalf("[]bool склеен неверно: %q", got)
	}

	if got := values.Get("pages"); got != "3" {
		t.Fatalf("uint8 разобран неверно: %q", got)
	}

	if values.Has("missing") {
		t.Fatal("пустой срез отправлять не нужно")
	}
}
