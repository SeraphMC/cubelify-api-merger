package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/SeraphMC/seraph-api-helpers/src/cubelify"
	"github.com/atotto/clipboard"
	"github.com/carlmjohnson/requests"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	apiConfigs      APIConfigs
	apiConfigsMutex sync.RWMutex
	configFile      = "config.json"
	mergeURL        = "http://localhost:3000/merger?sources={{sources}}&id={{id}}&name={{name}}"
)

var seraphBanner = `
 ____                       _     
/ ___|  ___ _ __ __ _ _ __ | |__  
\___ \ / _ \ '__/ _' | '_ \| '_ \ 
 ___) |  __/ | | (_| | |_) | | | |
|____/ \___|_|  \__,_| .__/|_| |_| 
                     |_|          
`

var (
	primary    = lipgloss.Color("#6C8EEF")
	secondary  = lipgloss.Color("#9ECBFF")
	accent     = lipgloss.Color("#FFD787")
	successCol = lipgloss.Color("#A6E3A1")
	errorCol   = lipgloss.Color("#F38BA8")
	textCol    = lipgloss.Color("#CDD6F4")
	muted      = lipgloss.Color("#7F849C")

	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(primary)
	subtitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(secondary)
	normalStyle    = lipgloss.NewStyle().Foreground(textCol)
	mutedStyle     = lipgloss.NewStyle().Foreground(muted).Italic(true)
	highlightStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	successStyle   = lipgloss.NewStyle().Foreground(successCol)
	errorStyle     = lipgloss.NewStyle().Foreground(errorCol)
	infoStyle      = lipgloss.NewStyle().Foreground(secondary)
)

func fetchData(customName, userAgent string, config APIConfig, requestParams map[string]string, results chan<- cubelify.CubelifyResponse, responseTimes chan<- time.Duration, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

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
	err := request.ToJSON(apiResponse).UserAgent(userAgent).CheckStatus(200).Fetch(context.Background())
	responseTime := time.Since(startTime)

	if err != nil {
		errChan <- fmt.Errorf("[%s]: %w", customName, err)
		return
	}

	results <- *apiResponse
	responseTimes <- responseTime
}

func copyToClipboard(s string) tea.Cmd {
	return func() tea.Msg {
		err := clipboard.WriteAll(s)
		return ClipboardMsg{Success: err == nil, Err: err}
	}
}

func main() {
	var err error
	apiConfigs, err = readAPIConfigs(configFile)
	if err != nil {
		fmt.Println("Error reading config:", err)
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
		userAgent := c.Get("User-Agent")

		apiConfigsMutex.RLock()
		for name, config := range apiConfigs {
			wg.Add(1)
			go fetchData(name, userAgent, config, requestParams, results, responseTimes, errChan, &wg)
		}
		apiConfigsMutex.RUnlock()

		wg.Wait()
		close(results)
		close(responseTimes)
		close(errChan)

		for err := range errChan {
			log.Println("Fetch error:", err)
		}

		builder := cubelify.NewCubelifyResponseBuilder()
		for result := range results {
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

	if _, err := tea.NewProgram(initialMenuModel(), tea.WithAltScreen()).Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}

	if err := app.Shutdown(); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}
}
