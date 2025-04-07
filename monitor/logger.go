package monitor

import (
	"io"
	"log"
	"os"
)

var logger *log.Logger

func initLogger() {
	logFile, err := os.OpenFile("monitor.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("❌ Failed to open log file: %v", err)
	}

	multi := io.MultiWriter(os.Stdout, logFile)
	logger = log.New(multi, "", log.Ldate|log.Ltime)
}

func logInfo(format string, args ...interface{}) {
	logger.Printf("[INFO] "+format, args...)
}

func logError(format string, args ...interface{}) {
	logger.Printf("[ERROR] "+format, args...)
}

func logFatal(format string, args ...interface{}) {
	logger.Fatalf("[FATAL] "+format, args...)
}
