package src

import (
	"os"
	"sync"

	"github.com/goccy/go-json"
)

var (
	ApiConfigs      APIConfigs
	ApiConfigsMutex sync.RWMutex
	ConfigFile      string
)

func init() {
	if path, err := os.Getwd(); err == nil {
		ConfigFile = path + string(os.PathSeparator) + "config.json"
	}
}

func ReadAPIConfigs() (APIConfigs, error) {
	data, err := os.ReadFile(ConfigFile)
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

	ApiConfigs = configs
	return configs, nil
}

func SaveAPIConfigs(configs APIConfigs) error {
	data, err := json.Marshal(configs)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, data, 0644)
}

func GetAPINames() []string {
	ApiConfigsMutex.RLock()
	defer ApiConfigsMutex.RUnlock()

	names := make([]string, 0, len(ApiConfigs))
	for name := range ApiConfigs {
		names = append(names, name)
	}
	return names
}
