package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-ping/ping"
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
	pinger, err := ping.NewPinger(host)
	if err != nil {
		logError("Ошибка создания пинга для %s: %v", host, err)
		return
	}

	pinger.Count = 3
	pinger.Timeout = 2 * time.Second
	pinger.SetPrivileged(true)

	if err = pinger.Run(); err != nil {
		msg := fmt.Sprintf("❌ Пинг не удался: %s — %v", host, err)
		logError(msg)
		sendTelegramMessage(msg)
		return
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		msg := fmt.Sprintf("🔴 Хост недоступен: %s (0 пакетов)", host)
		logError(msg)
		sendTelegramMessage(msg)
	} else {
		logInfo("✅ %s OK (%d/%d пакетов)", host, stats.PacketsRecv, stats.PacketsSent)
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
