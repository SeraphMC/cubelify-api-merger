package main

import (
	"os"

	"github.com/goccy/go-json"
)

func readAPIConfigs(filename string) (APIConfigs, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(APIConfigs), nil
		}
		return nil, err
	}

	var configs APIConfigs
	err = json.Unmarshal(data, &configs)
	if err != nil {
		return nil, err
	}

	return configs, nil
}

func saveAPIConfigs(filename string, configs APIConfigs) error {
	data, err := json.Marshal(configs)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func getAPINames() []string {
	apiConfigsMutex.RLock()
	defer apiConfigsMutex.RUnlock()

	names := make([]string, 0, len(apiConfigs))
	for name := range apiConfigs {
		names = append(names, name)
	}
	return names
}
