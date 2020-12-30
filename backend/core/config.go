package core

import (
	"io/fs"
	"log"
	"net"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/trezcool/masomo/fs"
)

var (
	Conf  conf
	build = "develop"
)

type (
	conf struct {
		Build                     string
		Env                       string
		Debug                     bool
		TestMode                  bool
		AppName                   string
		WorkDir                   string
		SecretKey                 string
		DefaultFromEmail          mail.Address
		FrontendBaseURL           string
		PasswordResetTimeoutDelta time.Duration
		SendgridApiKey            string
		RollbarToken              string
		Database                  dbConf
		Server                    srvConf
	}

	dbConf struct {
		Engine     string
		Name       string
		User       string
		Password   string
		Host       string
		Port       string
		DisableTLS bool
	}

	srvConf struct {
		Host                      string
		Port                      string
		DebugHost                 string
		ShutdownTimeout           time.Duration
		JWTExpirationDelta        time.Duration
		JWTRefreshExpirationDelta time.Duration
	}
)

func (dc dbConf) Address() string {
	return net.JoinHostPort(dc.Host, dc.Port)
}

func (sc srvConf) Address() string {
	return net.JoinHostPort(sc.Host, sc.Port)
}

func init() {
	v := viper.New()
	v.SetTypeByDefaultValue(true)

	env := os.Getenv("ENV") // DEV (local; default), TEST, QA, PROD
	if env == "" {
		env = "DEV"
	}
	v.SetEnvPrefix(env)

	// load .env if it exists (ignore if it does not)
	dotEnvPath := "config/" + strings.ToLower(env) + ".env"
	if file, err := appfs.FS.Open(dotEnvPath); err == nil {
		defer file.Close()
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
	v.SetDefault("workDir", getwd())
	v.SetDefault("secretKey", "poq5-wer)enb$+57=dz&uoxh2(h!x)#*c2(#yg4h^$cegm2emy")
	v.SetDefault("frontendBaseURL", "http://localhost:8080")
	v.SetDefault("passwordResetTimeoutDelta", 3*24*time.Hour)

	v.SetDefault("database.engine", "postgres")
	v.SetDefault("database.name", strings.ToLower(appName))
	v.SetDefault("database.user", "")
	v.SetDefault("database.password", "")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", "5432")
	v.SetDefault("database.disableTLS", true)

	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.port", "8000")
	v.SetDefault("server.debugHost", "localhost:9000")
	v.SetDefault("server.shutdownTimeout", 5*time.Second)
	v.SetDefault("server.jwtExpirationDelta", 7*24*time.Hour)
	v.SetDefault("server.jwtRefreshExpirationDelta", 4*time.Hour)

	v.SetDefault("sendgridApiKey", "")
	v.SetDefault("rollbarToken", "")
	// --------------------------------------------------------------------

	// check env vars and override defaults
	v.AutomaticEnv()

	if err := v.Unmarshal(&Conf); err != nil {
		log.Fatal(err)
	}

	Conf.DefaultFromEmail = mail.Address{
		Name:    v.GetString("appName"),
		Address: "noreply@" + v.GetString("server.host"),
	}
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

// todo: get rid of this once fs.FS is fully supported by all libs used
// getwd tries to find the project root "backend".
// go-test changes the working directory to the test package being run during tests... this breaks our code...
// see: https://stackoverflow.com/questions/23847003/golang-tests-and-working-directory
// this is a temporary fix for now :(
func getwd() string {
	root := "backend"
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "getting working directory"))
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
