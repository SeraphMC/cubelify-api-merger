package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/SeraphMC/seraph-api-helpers/src/cubelify"
	"github.com/carlmjohnson/requests"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"seraph.si/v2/api-merger/src"
)

func fetchData(customName, userAgent string, config src.APIConfig, requestParams map[string]string, results chan<- *cubelify.CubelifyResponse, responseTimes chan<- time.Duration, errChan chan<- error, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		if r := recover(); r != nil {
			log.Printf("Panic in fetchData for [%s]: %v", customName, r)
		}
	}()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	apiResponse := new(cubelify.CubelifyResponse)
	request := requests.URL(config.URL)

	for key, value := range config.Querystring {
		request.Param(key, fmt.Sprintf("%v", value))
	}

	if config.RequestParams != nil {
		for requestKey, configKey := range config.RequestParams {
			if value, ok := requestParams[requestKey]; ok {
				request.Param(configKey, value)
			}
		}
	} else {
		for key, value := range requestParams {
			request.Param(key, value)
		}
	}

	startTime := time.Now()
	err := request.ToJSON(apiResponse).Client(client).UserAgent(userAgent).CheckStatus(200).Fetch(context.Background())
	responseTime := time.Since(startTime)

	if err != nil {
		errChan <- fmt.Errorf("[%s]: %w", customName, err)
		return
	}

	results <- apiResponse
	responseTimes <- responseTime
}

func init() {
	_, err := src.ReadAPIConfigs()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	app := fiber.New(fiber.Config{
		AppName:               "API Merger by Seraph",
		GETOnly:               true,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
		DisableStartupMessage: true,
	})

	app.Get("/merger", func(c *fiber.Ctx) error {
		var wg sync.WaitGroup
		results := make(chan *cubelify.CubelifyResponse, len(src.ApiConfigs))
		responseTimes := make(chan time.Duration, len(src.ApiConfigs))
		errChan := make(chan error, len(src.ApiConfigs))

		requestParams := make(map[string]string)

		for key, value := range c.Request().URI().QueryArgs().All() {
			requestParams[string(key)] = string(value)
		}
		userAgent := utils.CopyString(c.Get("User-Agent"))

		src.ApiConfigsMutex.RLock()
		for name, config := range src.ApiConfigs {
			wg.Add(1)
			go fetchData(name, userAgent, config, requestParams, results, responseTimes, errChan, &wg)
		}
		src.ApiConfigsMutex.RUnlock()

		var fetchErrors []error
		for len(errChan) > 0 {
			select {
			case err := <-errChan:
				fetchErrors = append(fetchErrors, err)
			default:
				break
			}
		}

		wg.Wait()
		close(results)
		close(responseTimes)
		close(errChan)

		for _, err := range fetchErrors {
			log.Println("Error fetching from API:", err)
		}

		builder := cubelify.NewCubelifyResponseBuilder()
		for result := range results {
			if result == nil {
				continue
			}

			if result.Tags != nil {
				builder.AddTags(*result.Tags)
			}
			if result.Score != nil {
				builder.AddSniperScore(result.Score)
			}
		}

		return c.JSON(builder.Build())
	})

	go func() {
		if err := app.Listen(":3000"); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	if _, err := tea.NewProgram(src.InitialMenuModel(), tea.WithAltScreen()).Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}

	if err := app.Shutdown(); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}
}
