package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/protofire/filecoin-rpc-proxy/internal/updater"

	"github.com/protofire/filecoin-rpc-proxy/internal/cache"
	"github.com/protofire/filecoin-rpc-proxy/internal/matcher"

	"github.com/sirupsen/logrus"

	"github.com/protofire/filecoin-rpc-proxy/internal/metrics"

	"github.com/protofire/filecoin-rpc-proxy/internal/proxy"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"

	"github.com/protofire/filecoin-rpc-proxy/internal/config"
	"github.com/protofire/filecoin-rpc-proxy/internal/utils"
	"github.com/urfave/cli/v2"
)

var defaultConfigFileName = "proxy.yaml"

func getDefaultConfigFilePath() string {
	home, err := utils.GetUserHome()
	if err != nil {
		fmt.Println(err)
		home, err = os.Getwd()
		if err != nil {
			fmt.Println(err)
			home = "/"
		}
	}
	return path.Join(home, defaultConfigFileName)
}

func startCommand(c *cli.Context) error {
	configFile := c.String("config")
	if configFile == "" {
		configFile = getDefaultConfigFilePath()
	}
	if !utils.FileExists(configFile) {
		return fmt.Errorf("cannot find conf file file: %s", configFile)
	}
	conf, err := config.FromFile(configFile)
	if err != nil {
		return err
	}
	log := logger.InitLogger(conf.LogLevel, conf.LogPrettyPrint)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	cacher := proxy.NewResponseCache(
		cache.NewMemoryCacheFromConfig(conf),
		matcher.FromConfig(conf),
	)
	transportImp := proxy.NewTransport(cacher, log, conf.DebugHTTPRequest, conf.DebugHTTPResponse)

	updaterImp, err := updater.FromConfig(conf, cacher, log)
	if err != nil {
		return err
	}

	server, err := proxy.FromConfigWithTransport(conf, log, transportImp)
	if err != nil {
		return err
	}

	metrics.Register()

	handler := proxy.PrepareRoutes(conf, log, server)
	s := server.StartHTTPServer(handler)

	ctx, done := context.WithCancel(context.Background())
	go updaterImp.StartMethodUpdater(ctx, conf.UpdateCustomCachePeriod)
	go updaterImp.StartCacheUpdater(ctx, conf.UpdateUserCachePeriod)

	sig := <-stop
	log.Infof("Caught sig: %+v. Waiting process is being stopped...", sig)
	done()

	ctxUpdater, cancelUpdater := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
	defer cancelUpdater()

	if updaterImp.StopWithTimeout(ctxUpdater, conf.ShutdownTimeout) {
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
	app.Description = `
   Default config file is: ~/proxy.yaml
   Yaml format examples:

   ---
   proxy_url: http://test.com
   port: 8080
   cache_methods:
   - name: method
     cache_by_params: true
     params_for_request:
       - 1
       - one
       - two
     params_in_cache_by_id:
       - 1
       - 2

   ---
   proxy_url: http://test.com
   port: 8080
   cache_methods:
   - name: method
     cache_by_params: true
     params_for_request:
       - 1
       - one
       - two
     params_in_cache_by_name:
       - name1
       - name2
   `
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "config",
			Aliases:  []string{"c"},
			EnvVars:  []string{"RPC_PROXY_CONFIG_FILE"},
			Value:    getDefaultConfigFilePath(),
			Required: false,
			Usage:    "Config file. yaml format",
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
