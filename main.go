package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	InfoHandler "github.com/fudgydrs/valhalla-api/api/info"
	"github.com/sirupsen/logrus"

	"github.com/joho/godotenv"
)

type CustomLogFormatter struct{}

func (f *CustomLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	level := entry.Level.String()

	// Set log colors, using 256-color ANSI escape code
	var levelColor string
	switch entry.Level {
	case logrus.InfoLevel:
		levelColor = "\033[38;5;45m" // Green
	case logrus.DebugLevel:
		levelColor = "\033[34m" // Blue
	case logrus.WarnLevel:
		levelColor = "\033[33m" // Yellow
	case logrus.ErrorLevel:
		levelColor = "\033[31m" // Red
	case logrus.FatalLevel:
		levelColor = "\033[35m" // Magenta
	case logrus.PanicLevel:
		levelColor = "\033[36m" // Cyan
	default:
		levelColor = "\033[0m" // Reset color (for unknown levels)
	}

	logMessage := fmt.Sprintf("\n\033[38;5;180m%s\033[0m [%s%s\033[0m] %s\n",
		entry.Time.Format("2006-01-02 15:04:05"), // Timestamp in cream color
		levelColor,                               // Colorized log level
		level,                                    // Log level name
		entry.Message)                            // Log message

	return []byte(logMessage), nil
}

func main() {
	serverEnv := flag.String("server", "production", "Specify the server environment (local/production)")
	flag.Parse()

	var envFile string
	if *serverEnv == "local" {
		envFile = ".env.local"
	} else {
		envFile = ".env"
	}

	err := godotenv.Load(envFile)
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	debugMode := os.Getenv("DEBUG_MODE_ENABLED")
	if debugMode == "true" || debugMode == "1" {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.FatalLevel)
	}

	logrus.SetFormatter(&CustomLogFormatter{})

	logrus.Warning("program starting in debug mode...")

	http.HandleFunc("/api/info", InfoHandler.Handler)

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
