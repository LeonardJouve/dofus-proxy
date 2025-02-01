package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
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
const ZAAP_FILE = "C:/Users/%s/AppData/Local/Ankama/Dofus-dofus3/zaap.yml"

func editZaap(from string, to string) error {
	username := os.Getenv("USERNAME")
	file := fmt.Sprintf(ZAAP_FILE, username)
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	newContent := strings.Replace(string(content), from, to, 1)

	fileInfo, err := os.Stat(file)
	if err != nil {
		return err
	}

	err = os.WriteFile(file, []byte(newContent), fileInfo.Mode())
	if err != nil {
		return err
	}

	return nil
}

func RedirectZaap(configPort uint16) (func(), error) {
	redirectUrl := fmt.Sprintf("http://localhost:%d", configPort)
	return func() {
		editZaap(redirectUrl, CONFIG_URL)
	}, editZaap(CONFIG_URL, redirectUrl)
}

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
		return "", errors.New("no connection host")
	}

	re := regexp.MustCompile(`([a-zA-Z0-9.-]+:\d+)`)
	originalHost := re.FindString(config.ConnectionHosts[0])
	config.ConnectionHosts[0] = re.ReplaceAllString((config.ConnectionHosts[0]), fmt.Sprintf("localhost:%d", proxyPort))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(config)
	})

	go http.ListenAndServe(fmt.Sprintf(":%d", configPort), nil)

	fmt.Printf("Config listening on %d\n", configPort)

	return originalHost, nil
}
