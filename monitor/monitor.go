package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

type Config struct {
	BotToken      string   `json:"bot_token"`
	ChatID        string   `json:"chat_id"`
	CheckInterval int      `json:"check_interval_sec"`
	Hosts         []string `json:"hosts"`
}

var config Config

func Start(configFile string) {
	initLogger()

	if err := loadConfig(configFile); err != nil {
		logFatal("Ошибка загрузки конфигурации: %v", err)
	}

	logInfo("📡 Мониторинг запущен. Хосты: %v", config.Hosts)

	for {
		for _, host := range config.Hosts {
			go pingHost(host)
		}
		time.Sleep(time.Duration(config.CheckInterval) * time.Second)
	}
}

func loadConfig(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &config)
}

func pingHost(host string) {
	var out bytes.Buffer
	cmd := exec.Command("ping", "-c", "3", "-W", "2", host) // Linux/macOS
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	output := out.String()

	if err != nil {
		msg := fmt.Sprintf("🔴 Хост недоступен: %s\n%s", host, output)
		logError(msg)
		sendTelegramMessage(msg)
		return
	}

	if strings.Contains(output, "0 received") || strings.Contains(output, "100% packet loss") {
		msg := fmt.Sprintf("🔴 Потеря пакетов при пинге: %s\n%s", host, output)
		logError(msg)
		sendTelegramMessage(msg)
	} else {
		logInfo("✅ %s OK\n%s", host, output)
	}
}

func sendTelegramMessage(text string) {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.BotToken)
	data := url.Values{}
	data.Set("chat_id", config.ChatID)
	data.Set("text", text)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		logError("Ошибка создания запроса в Telegram: %v", err)
		return
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logError("Ошибка отправки сообщения в Telegram: %v", err)
		return
	}
	defer resp.Body.Close()
}
