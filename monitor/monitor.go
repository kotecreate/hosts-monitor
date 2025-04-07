package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
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
		logFatal("Error loading configuration: %v", err)
	}

	logInfo("📡 Monitoring started. Hosts: %v", config.Hosts)

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
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "3", "-w", "2000", host)
	} else {
		cmd = exec.Command("ping", "-c", "3", "-W", "2", host)
	}

	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()

	output := out.String()
	if runtime.GOOS == "windows" {
		output = decodeCP866ToUTF8(output)
	}

	if err != nil {
		msg := fmt.Sprintf("🔴 Host unavailable: %s\n%s", host, output)
		logError(msg)
		sendTelegramMessage(msg)
		return
	}

	if strings.Contains(output, "0 received") ||
		strings.Contains(output, "100% packet loss") ||
		strings.Contains(output, "Превышен интервал ожидания") {
		msg := fmt.Sprintf("🔴 Packet loss during ping: %s\n%s", host, output)
		logError(msg)
		sendTelegramMessage(msg)
	} else {
		logInfo("✅ %s OK\n%s", host, output)
	}
}

func decodeCP866ToUTF8(input string) string {
	decoder := charmap.CodePage866.NewDecoder()
	utf8Str, err := decoder.String(input)
	if err != nil {
		return input
	}
	return utf8Str
}

func sendTelegramMessage(text string) {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.BotToken)
	data := url.Values{}
	data.Set("chat_id", config.ChatID)
	data.Set("text", text)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		logError("Error creating request in Telegram: %v", err)
		return
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logError("Error sending message in Telegram: %v", err)
		return
	}
	defer resp.Body.Close()
}
