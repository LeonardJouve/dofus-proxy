package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Config struct {
	GameAppId                  int      `json:"gameAppId"`
	ConnectionHosts            []string `json:"connectionHosts"`
	BuildType                  string   `json:"buildType"`
	ChatAppId                  int      `json:"chatAppId"`
	ChatServerHost             string   `json:"chatServerHost"`
	ChatServerPort             int      `json:"chatServerPort"`
	VersionFileUrl             string   `json:"versionFileUrl"`
	HaapiAnkamaUrl             string   `json:"haapiAnkamaUrl"`
	HaapiDofusUrl              string   `json:"haapiDofusUrl"`
	ShopiDofusUrl              string   `json:"shopiDofusUrl"`
	WebShopDofusUrl            string   `json:"webShopDofusUrl"`
	GamesActivityDescriptorUrl string   `json:"gamesActivityDescriptorUrl"`
	AvatarUrlFormat            string   `json:"avatarUrlFormat"`
	DofusWebsiteUrl            string   `json:"dofusWebsiteUrl"`
}

const CONFIG_URL = "https://dofus2.cdn.ankama.com/config/release_windows.json"

func Run(proxyPort uint16, configPort uint16) (string, error) {
	response, err := http.Get(CONFIG_URL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var config Config
	err = json.Unmarshal(body, &config)
	if err != nil {
		return "", err
	}

	if len(config.ConnectionHosts) == 0 {
		return "", errors.New("no connection hosts field")
	}
	originalHost := config.ConnectionHosts[0]
	// TODO: parse original host
	config.ConnectionHosts = []string{fmt.Sprintf("http://localhost:%d", proxyPort)}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(config)
	})

	http.ListenAndServe(fmt.Sprintf(":%d", configPort), nil)

	fmt.Printf("Config listening on %d\n", configPort)

	return originalHost, nil
}
