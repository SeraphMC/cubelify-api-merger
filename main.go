package main

import (
	"context"
	"fmt"
	"github.com/SeraphMC/seraph-api-helpers/src/cubelify"
	"github.com/carlmjohnson/requests"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"log"
	"os"
	"sync"
	"time"
)

type APIConfigs map[string]APIConfig

type APIConfig struct {
	// Base URL
	URL string `json:"url"`
	// Query string parameters (optional)
	Querystring map[string]interface{} `json:"querystring"`
	// Remap generic request parameters to custom ones (optional)
	RequestParams map[string]string `json:"request_params"`
}

func fetchData(customName string, config APIConfig, requestParams map[string]string, results chan<- cubelify.CubelifyResponse, responseTimes chan<- time.Duration, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	apiResponse := new(cubelify.CubelifyResponse)
	request := requests.URL(config.URL)

	if config.Querystring != nil {
		for key, value := range config.Querystring {
			request.Param(key, fmt.Sprintf("%v", value))
		}
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
	err := request.ToJSON(apiResponse).CheckStatus(200).Fetch(context.Background())
	endTime := time.Now()
	responseTime := endTime.Sub(startTime)

	if err != nil {
		errChan <- fmt.Errorf("[%s]: %w", customName, err)
		return
	}

	results <- *apiResponse
	responseTimes <- responseTime
}

func main() {
	configFile := "config.json"
	apiConfigs, err := readAPIConfigs(configFile)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return
	}

	app := fiber.New(fiber.Config{
		AppName:     "API Merger by Seraph",
		GETOnly:     true,
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	app.Get("/merger", func(c *fiber.Ctx) error {
		var wg sync.WaitGroup
		results := make(chan cubelify.CubelifyResponse, len(apiConfigs))
		responseTimes := make(chan time.Duration, len(apiConfigs))
		errChan := make(chan error, len(apiConfigs))

		requestParams := make(map[string]string)
		c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
			requestParams[string(key)] = string(value)
		})

		for customName, config := range apiConfigs {
			wg.Add(1)
			go fetchData(customName, config, requestParams, results, responseTimes, errChan, &wg)
		}

		wg.Wait()
		close(results)
		close(responseTimes)
		close(errChan)

		individualResponseTimes := make(map[string]time.Duration)
		for customName := range apiConfigs {
			responseTime := <-responseTimes
			individualResponseTimes[customName] = responseTime
		}

		for err := range errChan {
			fmt.Println("Error:", err)
		}

		for name, rt := range individualResponseTimes {
			fmt.Printf("[%s] responded in %dms with URI: %s\n", name, rt.Milliseconds(), apiConfigs[name].URL)
		}

		mergedResults := cubelify.NewCubelifyResponseBuilder()
		for result := range results {
			if result.Tags != nil {
				mergedResults.AddTags(*result.Tags)
			}
			if result.Score != nil {
				mergedResults.AddSniperScore(result.Score)
			}
		}

		return c.JSON(mergedResults.Build())
	})

	log.Fatal(app.Listen(":3000"))
}

func readAPIConfigs(filename string) (APIConfigs, error) {
	var configs APIConfigs
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(file, &configs)
	return configs, err
}
