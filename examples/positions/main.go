// Позиции сайта в Яндексе по списку запросов.
//
// Запуск: JSONSEO_KEY=ваш_ключ go run ./examples/positions
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	jsonseo "github.com/jsonseo/go-sdk"
)

func main() {
	client, err := jsonseo.New(os.Getenv("JSONSEO_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	const domain = "example.com"
	queries := []string{"купить ноутбук", "ноутбук недорого"}

	for _, query := range queries {
		// break_domain останавливает поиск: платить за страницы ниже незачем.
		serp, err := client.Yandex(context.Background(), query, jsonseo.Params{
			"region":       213,
			"pages":        10,
			"break_domain": domain,
		})
		if err != nil {
			fmt.Printf("%s: ошибка — %v\n", query, err)

			continue
		}

		position := 0

		for index, result := range serp.Results {
			// Сравнивать домены напрямую нельзя: выдача отдаёт их с
			// поддоменом, и example.com не совпал бы с www.example.com.
			found := strings.ToLower(result.Domain)
			if found == domain || strings.HasSuffix(found, "."+domain) {
				position = index + 1

				break
			}
		}

		if position == 0 {
			fmt.Printf("%s: не найден в топ-%d\n", query, len(serp.Results))
		} else {
			fmt.Printf("%s: %d\n", query, position)
		}
	}
}
