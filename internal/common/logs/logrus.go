package logs

import (
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func Init() {
	SetFormatter(logrus.StandardLogger())

	logrus.SetLevel(logrus.DebugLevel)
}

func SetFormatter(logger *logrus.Logger) {
	// Example //^ text-formatted log entry: 2023-08-05T10:00:00Z [INFO] Log message custom_field=value
	// Example // * JSON-formatted log entry: {"time":"2023-08-05T10:00:00Z","severity":"info","message":"Log message","custom_field":"value"}

	logger.SetFormatter(&logrus.JSONFormatter{ //
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "time",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
	})

	if isLocalEnv, _ := strconv.ParseBool(os.Getenv("LOCAL_ENV")); isLocalEnv {
		logger.SetFormatter(&prefixed.TextFormatter{
			ForceFormatting: true,
		})
	}
}
