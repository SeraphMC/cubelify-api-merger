package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/SeraphMC/seraph-api-helpers/src/cubelify"
	"github.com/carlmjohnson/requests"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type APIConfigs map[string]APIConfig

type APIConfig struct {
	URL           string                 `json:"url"`
	Querystring   map[string]interface{} `json:"querystring"`
	RequestParams map[string]string      `json:"request_params"`
}

var apiConfigs APIConfigs
var apiConfigsMutex sync.RWMutex
var configFile = "config.json"

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
	var err error
	apiConfigs, err = readAPIConfigs(configFile)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		apiConfigs = make(APIConfigs)
	}

	app := fiber.New(fiber.Config{
		AppName:               "API Merger by Seraph",
		GETOnly:               true,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
		DisableStartupMessage: true,
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

		apiConfigsMutex.RLock()
		for customName, config := range apiConfigs {
			wg.Add(1)
			go fetchData(customName, config, requestParams, results, responseTimes, errChan, &wg)
		}
		apiConfigsMutex.RUnlock()

		wg.Wait()
		close(results)
		close(responseTimes)
		close(errChan)

		individualResponseTimes := make(map[string]time.Duration)
		apiConfigsMutex.RLock()
		for customName := range apiConfigs {
			responseTime := <-responseTimes
			individualResponseTimes[customName] = responseTime
		}
		apiConfigsMutex.RUnlock()

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

	go func() {
		fmt.Println("\nStarting web server on port 3000")
		if err := app.Listen(":3000"); err != nil {
			log.Fatalf("Error starting web server: %v", err)
		}
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\nAPI Merger by Seraph")
		fmt.Println("Options:")
		fmt.Println("1. Add API Configuration")
		fmt.Println("2. Exit")
		fmt.Print("Enter your choice: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("An error occurred:", err)
			if errors.Is(err, os.ErrClosed) {
				fmt.Println("Exiting due to closed input.")
				break
			}
			continue
		}
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			addAPIConfig(reader)
		case "2":
			fmt.Println("Exiting...")
			if err := app.Shutdown(); err != nil {
				log.Printf("Error shutting down server: %v", err)
			}
			return
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}

func addAPIConfig(reader *bufio.Reader) {
	fmt.Print("Enter API Name: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading name:", err)
		return
	}
	name = strings.ReplaceAll(strings.TrimSpace(name), " ", "-")

	fmt.Print("Enter API URL: ")
	urlStr, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading URL:", err)
		return
	}
	urlStr = strings.TrimSpace(urlStr)

	if name == "" || urlStr == "" {
		fmt.Println("Both name and URL are required.")
		return
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		fmt.Println("Invalid URL:", err)
		return
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		fmt.Println("Invalid URL format. Must start with http:// or https://")
		return
	}

	queryParams := make(map[string]interface{})
	for key, values := range parsedURL.Query() {
		if len(values) > 1 {
			queryParams[key] = values
		} else {
			queryParams[key] = values[0]
		}
	}

	apiConfigsMutex.Lock()
	if _, exists := apiConfigs[name]; exists {
		apiConfigsMutex.Unlock()
		fmt.Printf("API configuration with name '%s' already exists.\n", name)
		return
	}

	apiConfigs[name] = APIConfig{
		URL:         fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path),
		Querystring: queryParams,
	}
	apiConfigsMutex.Unlock()

	err = saveAPIConfigs(configFile, apiConfigs)
	if err != nil {
		fmt.Println("Failed to save configuration:", err)
		return
	}

	fmt.Printf("API configuration added with name '%s' and URL '%s'.\n", name, urlStr)
}

func readAPIConfigs(filename string) (APIConfigs, error) {
	var configs APIConfigs
	file, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(APIConfigs), nil
		}
		return nil, err
	}
	err = json.Unmarshal(file, &configs)
	return configs, err
}

func saveAPIConfigs(filename string, configs APIConfigs) error {
	file, err := json.Marshal(configs)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, file, 0644)
}
