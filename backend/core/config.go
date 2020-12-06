package core

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var Conf *viper.Viper

func init() {
	Conf = viper.New()

	// defaults
	Conf.SetTypeByDefaultValue(true)
	Conf.SetDefault("debug", true)
	Conf.SetDefault("appName", "Masomo")
	Conf.SetDefault("secretKey", "poq5-wer)enb$+57=dz&uoxh2(h!x)#*c2(#yg4h^$cegm2emy")
	Conf.SetDefault("defaultFromEmail", "noreply@localhost")
	Conf.SetDefault("jwtExpirationDelta", 7*24*time.Hour)
	Conf.SetDefault("jwtRefreshExpirationDelta", 4*time.Hour)
	Conf.SetDefault("passwordResetTimeoutDelta", 3*24*time.Hour)

	env := os.Getenv("ENV") // DEV (local; default), TEST, QA, PROD
	switch env {
	case "":
		env = "DEV"
	case strings.ToUpper("TEST"):
		Conf.SetDefault("testMode", true)
	}
	Conf.SetEnvPrefix(env)

	// load .env if it exists (ignore if it does not)
	dotEnvPath := filepath.Join(Getwd(), "config", ".env."+strings.ToLower(env))
	if _, err := os.Stat(dotEnvPath); err == nil {
		if err := godotenv.Load(dotEnvPath); err != nil {
			log.Fatalf("config.godotenv(%s): %v", dotEnvPath, err)
		}
	} else if !os.IsNotExist(err) {
		log.Fatalf("config.os.Stat(%s): %v", dotEnvPath, err)
	}
	Conf.AutomaticEnv()
}
