package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/cache"
	"github.com/protofire/filecoin-rpc-proxy/internal/config"
	"github.com/protofire/filecoin-rpc-proxy/internal/logger"
	"github.com/protofire/filecoin-rpc-proxy/internal/matcher"
	"github.com/protofire/filecoin-rpc-proxy/internal/metrics"
	"github.com/protofire/filecoin-rpc-proxy/internal/proxy"
	"github.com/protofire/filecoin-rpc-proxy/internal/updater"
	"github.com/protofire/filecoin-rpc-proxy/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var defaultConfigFileName = "config.yaml"

func getDefaultConfigFilePath() string {
	home, err := utils.GetUserHome()
	if err != nil {
		home, err = os.Getwd()
		if err != nil {
			home = "/"
		}
	}
	return path.Join(home, defaultConfigFileName)
}

func getConfigExample() (string, error) {
	c, err := base64.StdEncoding.DecodeString(ConfigExample)
	if err != nil {
		return "", err
	}
	return string(c), nil
}

func startCommand(c *cli.Context) error {
	configFile := c.String("config")
	if configFile == "" {
		configFile = getDefaultConfigFilePath()
	}
	if !utils.FileExists(configFile) {
		return fmt.Errorf("cannot find conf file file: %s", configFile)
	}
	conf, err := config.FromFile(configFile, config.CmdLineParams{
		JWTSecret: c.String("jwt-secret"),
		ProxyURL:  c.String("proxy-url"),
		RedisURI:  c.String("redis-uri"),
	})
	if err != nil {
		return err
	}
	jwtSecret := c.String("jwt-secret")
	proxyURL := c.String("proxy-url")
	if jwtSecret != "" {
		conf.JWTSecretBase64 = jwtSecret
	}
	if proxyURL != "" {
		conf.ProxyURL = proxyURL
	}
	log := logger.InitLogger(conf.LogLevel, conf.LogPrettyPrint)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	ctx, done := context.WithCancel(context.Background())

	cacheImpl, err := cache.FromConfig(ctx, conf)
	if err != nil {
		done()
		return err
	}

	cacher := proxy.NewResponseCache(
		cacheImpl,
		matcher.FromConfig(conf),
	)
	transportImp := proxy.NewTransport(cacher, log, conf.DebugHTTPRequest, conf.DebugHTTPResponse)

	updaterImp, err := updater.FromConfig(conf, cacher, log)
	if err != nil {
		done()
		return err
	}

	server, err := proxy.FromConfigWithTransport(conf, log, transportImp)
	if err != nil {
		done()
		return err
	}

	defer func() {
		done()
		_ = server.Close()
	}()

	metrics.Register()

	handler := proxy.PrepareRoutes(conf, log, server)
	s := server.StartHTTPServer(handler)

	go updaterImp.StartMethodUpdater(ctx, conf.UpdateCustomCachePeriod)
	go updaterImp.StartCacheUpdater(ctx, conf.UpdateUserCachePeriod)

	sig := <-stop
	log.Infof("Caught sig: %+v. Waiting process is being stopped...", sig)
	done()
	if err := cacheImpl.Close(); err != nil {
		log.Error(err)
	}

	ctxUpdater, cancelUpdater := context.WithTimeout(context.Background(), time.Duration(conf.ShutdownTimeout)*time.Second)
	defer cancelUpdater()

	if updaterImp.StopWithTimeout(ctxUpdater, 2) {
		log.Info("Shut down server gracefully")
	} else {
		log.Info("Shut down server forcibly")
	}

	stopTimeout := 2
	ctxServer, cancelServer := context.WithTimeout(context.Background(), time.Duration(stopTimeout)*time.Second)
	defer cancelServer()
	if err = s.Shutdown(ctxServer); err != nil {
		log.Errorf("Could not stop server within %d seconds: %v", stopTimeout, err)
	} else {
		log.Info("Server has been stopped successfully")
	}

	return err
}

func prepareCliApp() *cli.App {
	configExample, err := getConfigExample()
	if err != nil {
		configExample = ""
	}
	app := cli.NewApp()
	app.Version = Version
	app.HideHelp = false
	app.HideVersion = false
	app.Authors = []*cli.Author{{
		Name:  "Igor Nemilentsev",
		Email: "trezorg@gmail.com",
	}}
	app.Usage = "JSON PRC cached proxy"
	app.EnableBashCompletion = true
	app.Action = startCommand
	app.Description = fmt.Sprintf(`
Default config file is: ~/config.yaml
Config file example:

---
%s`,
		configExample,
	)
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "config",
			Aliases:  []string{"c"},
			EnvVars:  []string{"PROXY_CONFIG_FILE"},
			Value:    getDefaultConfigFilePath(),
			Required: false,
			Usage:    "Config file. yaml format",
		},
		&cli.StringFlag{
			Name:     "jwt-secret",
			Aliases:  []string{"s"},
			EnvVars:  []string{"PROXY_JWT_SECRET"},
			Required: false,
			Usage:    "JWT secret in base64 format",
		},
		&cli.StringFlag{
			Name:     "proxy-url",
			Aliases:  []string{"p"},
			EnvVars:  []string{"PROXY_URL"},
			Required: false,
			Usage:    "JWT proxy url",
		},
		&cli.StringFlag{
			Name:     "redis-uri",
			Aliases:  []string{"r"},
			EnvVars:  []string{"REDIS_URI"},
			Required: false,
			Usage:    "Redis URI",
		},
	}

	return app
}

func main() {
	app := prepareCliApp()
	err := app.Run(os.Args)
	if err != nil {
		logrus.Errorf("Cannot initialize application: %+v", err)
		os.Exit(1)
	}
}
