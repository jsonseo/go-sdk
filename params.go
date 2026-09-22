package jsonseo

import (
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Params — необязательные параметры метода.
//
// Запись открыта не для удобства: вертикали принимают и родные параметры
// поисковиков — tbs, isize, qft, — перечислить которые нельзя, они меняются
// вместе с поисковиками.
//
//	client.Yandex(ctx, "купить ноутбук", jsonseo.Params{"region": 213, "ads": true})
type Params map[string]any

// merge собирает несколько наборов в один: так вызов принимает и ноль
// наборов, и несколько.
func merge(sets []Params) Params {
	if len(sets) == 0 {
		return Params{}
	}

	merged := make(Params)

	for _, set := range sets {
		for name, value := range set {
			merged[name] = value
		}
	}

	return merged
}

// encode приводит параметры к тому виду, в каком их ждёт форма запроса.
func encode(params Params) (url.Values, error) {
	values := url.Values{}

	// Имена сортируются, чтобы тело запроса не менялось от прогона к прогону:
	// иначе его нельзя сравнить ни в тесте, ни в логе.
	names := make([]string, 0, len(params))
	for name := range params {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		value := params[name]

		if value == nil {
			continue
		}

		// Срез любого типа, кроме строки: []int64 из базы и []float64 из
		// расчёта должны работать наравне с []string.
		if kind := reflect.ValueOf(value).Kind(); kind == reflect.Slice || kind == reflect.Array {
			items := reflect.ValueOf(value)

			if items.Len() == 0 {
				continue
			}

			parts := make([]string, items.Len())

			for i := 0; i < items.Len(); i++ {
				part, err := scalar(items.Index(i).Interface(), fmt.Sprintf("%s[%d]", name, i))
				if err != nil {
					return nil, err
				}

				parts[i] = part
			}

			values.Set(name, strings.Join(parts, separatorFor(name)))

			continue
		}

		part, err := scalar(value, name)
		if err != nil {
			return nil, err
		}

		values.Set(name, part)
	}

	return values, nil
}

// separatorFor: фразы склеиваются переводом строки, запятая в них встречается.
func separatorFor(name string) string {
	if name == "phrases" {
		return "\n"
	}

	return ","
}

// scalar смотрит на вид значения, а не на точный тип: иначе именованные
// типы вроде `type Device string` пришлось бы приводить руками.
func scalar(value any, name string) (string, error) {
	if value == nil {
		return "", fmt.Errorf("%w: значение параметра %s не задано", ErrInvalidArgument, name)
	}

	switch reflect.ValueOf(value).Kind() {
	case reflect.String:
		return reflect.ValueOf(value).String(), nil
	case reflect.Bool:
		if reflect.ValueOf(value).Bool() {
			return "1", nil
		}

		return "0", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(reflect.ValueOf(value).Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(reflect.ValueOf(value).Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(reflect.ValueOf(value).Float(), 'f', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(reflect.ValueOf(value).Float(), 'f', -1, 64), nil
	}

	if stringer, ok := value.(fmt.Stringer); ok {
		return stringer.String(), nil
	}

	return "", fmt.Errorf(
		"%w: значение параметра %s должно быть строкой, числом, флагом или срезом таких значений, получено %T",
		ErrInvalidArgument, name, value,
	)
}
