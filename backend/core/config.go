package core

import (
	"log"
	"net"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var Conf conf

type (
	conf struct {
		Debug                     bool
		TestMode                  bool
		AppName                   string
		WorkDir                   string
		SecretKey                 string
		DefaultFromEmail          mail.Address
		SendgridApiKey            string
		FrontendBaseURL           string
		PasswordResetTimeoutDelta time.Duration
		Database                  dbConf
		Server                    srvConf
		// Version // git version todo
	}

	dbConf struct {
		Engine     string
		Name       string
		User       string
		Password   string
		Host       string
		DisableTLS bool
	}

	srvConf struct {
		Host                      string
		ShutdownTimeout           time.Duration
		JWTExpirationDelta        time.Duration
		JWTRefreshExpirationDelta time.Duration
	}
)

func init() {
	v := viper.New()
	v.SetTypeByDefaultValue(true)

	env := os.Getenv("ENV") // DEV (local; default), TEST, QA, PROD
	if env == "" {
		env = "DEV"
	} else if strings.ToUpper(env) == "TEST" {
		v.SetDefault("testMode", true)
	}
	v.SetEnvPrefix(env)

	// load .env if it exists (ignore if it does not)
	wd := getwd()
	v.SetDefault("workDir", wd)
	dotEnvPath := filepath.Join(wd, "config", ".env."+strings.ToLower(env))
	if _, err := os.Stat(dotEnvPath); err == nil {
		if err := godotenv.Load(dotEnvPath); err != nil {
			log.Fatalf("config.godotenv(%s): %v", dotEnvPath, err)
		}
	} else if !os.IsNotExist(err) {
		log.Fatalf("config.os.Stat(%s): %v", dotEnvPath, err)
	}

	// ----------------------------- defaults ----------------------------
	appName := "Masomo"
	v.SetDefault("debug", true)
	v.SetDefault("appName", appName)
	v.SetDefault("secretKey", "poq5-wer)enb$+57=dz&uoxh2(h!x)#*c2(#yg4h^$cegm2emy")
	v.SetDefault("frontendBaseURL", "http://localhost:8080")
	v.SetDefault("passwordResetTimeoutDelta", 3*24*time.Hour)

	v.SetDefault("dbEngine", "postgres")
	v.SetDefault("dbName", strings.ToLower(appName))
	v.SetDefault("dbHost", "localhost")
	v.SetDefault("dbPort", "5432")
	v.SetDefault("dbDisableTLS", true)

	v.SetDefault("srvHost", "localhost")
	v.SetDefault("srvShutdownTimeout", 5*time.Second)
	v.SetDefault("srvJwtExpirationDelta", 7*24*time.Hour)
	v.SetDefault("srvJwtRefreshExpirationDelta", 4*time.Hour)
	// -------------------------------------------------------------------

	// check env vars and override defaults
	v.AutomaticEnv()

	setConfig(v)
}

func setConfig(v *viper.Viper) {
	Conf = conf{
		Debug:                     v.GetBool("debug"),
		TestMode:                  v.GetBool("testMode"),
		AppName:                   v.GetString("appName"),
		WorkDir:                   v.GetString("workDir"),
		SecretKey:                 v.GetString("secretKey"),
		SendgridApiKey:            v.GetString("sendgridApiKey"),
		FrontendBaseURL:           v.GetString("frontendBaseURL"),
		PasswordResetTimeoutDelta: v.GetDuration("passwordResetTimeoutDelta"),
		DefaultFromEmail: mail.Address{
			Name:    v.GetString("appName"),
			Address: "noreply@" + v.GetString("serverHost"),
		},
		Database: dbConf{
			Engine:     v.GetString("dbEngine"),
			Name:       v.GetString("dbName"),
			User:       v.GetString("dbUser"),
			Password:   v.GetString("dbPassword"),
			Host:       net.JoinHostPort(v.GetString("dbHost"), v.GetString("dbPort")),
			DisableTLS: v.GetBool("dbDisableTLS"),
		},
		Server: srvConf{
			Host:                      v.GetString("srvHost"),
			ShutdownTimeout:           v.GetDuration("srvShutdownTimeout"),
			JWTExpirationDelta:        v.GetDuration("srvJwtExpirationDelta"),
			JWTRefreshExpirationDelta: v.GetDuration("srvJwtRefreshExpirationDelta"),
		},
	}
}

// getwd tries to find the project root "backend".
// go-test changes the working directory to the test package being run during tests... this breaks our code...
// see: https://stackoverflow.com/questions/23847003/golang-tests-and-working-directory
// this is a temporary fix for now :(
func getwd() string {
	root := "backend"
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	currDir := wd
	for {
		if fi, err := os.Stat(currDir); err == nil {
			dirBase := filepath.Base(currDir)
			if dirBase == root && fi.IsDir() {
				return currDir
			}
		}
		newDir := filepath.Dir(currDir)
		if newDir == string(os.PathSeparator) || newDir == currDir {
			log.Fatal("project root not found")
		}
		currDir = newDir
	}
}
