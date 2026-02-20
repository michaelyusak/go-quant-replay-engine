package log

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	logLevels = map[string]logrus.Level{
		"DEBUG": logrus.DebugLevel,
		"INFO":  logrus.InfoLevel,
		"WARN":  logrus.WarnLevel,
		"ERROR": logrus.ErrorLevel,
	}
)

func SetupLogger(logLevel string, dir string) error {
	logrusLogLevel, ok := logLevels[logLevel]
	if !ok {
		return fmt.Errorf("[log][logrus][Init] invalid log level: %s", logLevel)
	}

	logrus.SetLevel(logrusLogLevel)

	if logrusLogLevel == logrus.DebugLevel {
		logrus.SetReportCaller(true)
	}

	now := time.Now()
	year, month, date := now.Date()

	dir += fmt.Sprintf("/%v-%s-%v", year, month.String(), date)

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("[log][logrus][Init][os.MkdirAll] %w", err)
	}

	if err := os.Chmod(dir, 0755); err != nil {
		return fmt.Errorf("[log][logrus][Init][os.Chmod] %w", err)
	}

	fileName := fmt.Sprintf("%s/%v.log", dir, now.Unix())

	// Open log file
	file, err := os.OpenFile(
		fileName,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		return fmt.Errorf("[log][logrus][Init][os.OpenFile] %w", err)
	}

	// Write logs to both stdout and file
	mw := io.MultiWriter(os.Stdout, file)
	logrus.SetOutput(mw)

	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	return nil
}
