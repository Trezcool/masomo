package core

import (
	"io/fs"
	"log"
	"net"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/trezcool/masomo/fs"
)

var build = "develop"

type (
	Config struct {
		Build                string
		Env                  string
		Debug                bool
		TestMode             bool
		AppName              string
		SecretKey            string
		FrontendBaseURL      string
		PasswordResetTimeout time.Duration
		SendgridApiKey       string
		RollbarToken         string
		Database             dbConf
		Server               srvConf
	}

	dbConf struct {
		Engine        string
		Name          string
		User          string
		Password      string
		AdminUser     string
		AdminPassword string
		Host          string
		Port          string
		DisableTLS    bool
	}

	srvConf struct {
		Host                 string
		Port                 string
		DebugHost            string
		ShutdownTimeout      time.Duration
		JWTExpiration        time.Duration
		JWTRefreshExpiration time.Duration
	}
)

func (c Config) DefaultFromEmail() mail.Address {
	return mail.Address{
		Name:    c.AppName,
		Address: "noreply@" + c.Server.Host,
	}
}

func (dc dbConf) Address() string {
	return net.JoinHostPort(dc.Host, dc.Port)
}

func (sc srvConf) Address() string {
	return net.JoinHostPort(sc.Host, sc.Port)
}

// NewConfig returns the application's Config instance
func NewConfig() *Config {
	v := viper.New()
	v.SetTypeByDefaultValue(true)

	env := os.Getenv("ENV") // DEV (local; default), TEST, QA, PROD
	if env == "" {
		env = "DEV"
	}
	v.SetEnvPrefix(env)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))

	// load .env if it exists (ignore if it does not)
	dotEnvPath := "config/" + strings.ToLower(env) + ".env"
	if file, err := appfs.FS.Open(dotEnvPath); err == nil {
		defer func() { _ = file.Close() }()
		if err != loadEnvFile(file) {
			log.Fatalf("%+v", errors.Wrap(err, "loading "+dotEnvPath))
		}
	} else if !os.IsNotExist(err) {
		log.Fatalf("%+v", errors.Wrap(err, "checking if "+dotEnvPath+" exists"))
	}

	// ----------------------------- defaults ----------------------------
	appName := "Masomo"
	v.SetDefault("build", build)
	v.SetDefault("env", strings.ToLower(env))
	v.SetDefault("debug", true)
	v.SetDefault("testMode", strings.EqualFold(env, "TEST"))
	v.SetDefault("appName", appName)
	v.SetDefault("secretKey", "poq5-wer)enb$+57=dz&uoxh2(h!x)#*c2(#yg4h^$cegm2emy")
	v.SetDefault("frontendBaseURL", "http://localhost:8080")
	v.SetDefault("passwordResetTimeout", 3*24*time.Hour)
	v.SetDefault("sendgridApiKey", "")
	v.SetDefault("rollbarToken", "")

	v.SetDefault("database.engine", "postgres")
	v.SetDefault("database.name", strings.ToLower(appName))
	v.SetDefault("database.user", "")
	v.SetDefault("database.password", "")
	v.SetDefault("database.adminUser", "")
	v.SetDefault("database.adminPassword", "")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", "5432")
	v.SetDefault("database.disableTLS", true)

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", "8000")
	v.SetDefault("server.debugHost", "0.0.0.0:9000")
	v.SetDefault("server.shutdownTimeout", 5*time.Second)
	v.SetDefault("server.jwtExpiration", 7*24*time.Hour)
	v.SetDefault("server.jwtRefreshExpiration", 4*time.Hour)
	// --------------------------------------------------------------------

	// check env vars and override defaults
	v.AutomaticEnv()

	conf := new(Config)
	if err := v.Unmarshal(&conf); err != nil {
		log.Fatal(err)
	}

	if conf.Debug {
		log.Printf("\n\nConf: %v\n\n", v.AllSettings())
	}
	return conf
}

// todo: get rid of this once `gotdotenv` supports new fs.FS
func loadEnvFile(file fs.File) error {
	envMap, err := godotenv.Parse(file)
	if err != nil {
		return err
	}

	currentEnv := map[string]bool{}
	rawEnv := os.Environ()
	for _, rawEnvLine := range rawEnv {
		key := strings.Split(rawEnvLine, "=")[0]
		currentEnv[key] = true
	}

	for key, value := range envMap {
		if !currentEnv[key] {
			_ = os.Setenv(key, value)
		}
	}
	return nil
}
