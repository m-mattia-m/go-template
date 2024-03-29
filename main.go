package main

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	sentrylogrus "github.com/getsentry/sentry-go/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"message-proxy/internal/api"
	"message-proxy/internal/service"
	"net/http"
	"time"
)

func main() {
	err := initConfig()
	if err != nil {
		log.Fatalln("Error reading config file, %s", err)
	}

	err = initLogger()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	svc := service.NewClient()

	log.Info("info for sentry")

	router := api.Router(svc)
	err = router.Run(fmt.Sprintf(":%s", viper.GetString("server.port")))
	if err != nil {
		log.Fatalf("Error starting api: %s", err)
	}
}

func initConfig() error {
	viper.AddConfigPath("./configs/")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	return nil
}

func initLogger() error {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.SetOutput(log.StandardLogger().Out)

	sentryLevels := []log.Level{log.ErrorLevel, log.FatalLevel, log.PanicLevel, log.InfoLevel}

	fmt.Println(fmt.Sprintf("DNS: %s", viper.GetString("logging.sentry.dsn")))
	sentryHook, err := sentrylogrus.New(sentryLevels, sentry.ClientOptions{
		Dsn: viper.GetString("logging.sentry.dsn"),
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// TODO: add sentry-tags
			//  event.Tags = map[string]string{}

			event.Environment = viper.GetString("app.env")

			if hint.Context != nil {
				if req, ok := hint.Context.Value(sentry.RequestContextKey).(*http.Request); ok {
					fmt.Println(req)
				}
			}
			if viper.GetBool("logging.enable.console") {
				fmt.Println(
					struct {
						timestamp string
						eventId   string
						message   string
					}{
						timestamp: event.Timestamp.String(),
						eventId:   fmt.Sprintf("%v", event.EventID),
						message:   event.Message,
					})
			}
			return event
		},
		Debug:            viper.GetString("app.env") == "PROD",
		AttachStacktrace: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create sentry-logrus-hook: %v", err)
	}
	defer sentryHook.Flush(5 * time.Second)
	log.AddHook(sentryHook)

	log.RegisterExitHandler(func() { sentryHook.Flush(5 * time.Second) })
	return nil
}
