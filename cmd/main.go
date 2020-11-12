package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/protofire/filecoin-rpc-proxy/internal/metrics"

	"github.com/protofire/filecoin-rpc-proxy/internal/proxy"

	"github.com/protofire/filecoin-rpc-proxy/internal/logger"

	config2 "github.com/protofire/filecoin-rpc-proxy/internal/config"
	"github.com/protofire/filecoin-rpc-proxy/internal/utils"
	"github.com/urfave/cli/v2"
)

var defaultConfigFile = "proxy.yaml"

func startCommand(c *cli.Context) error {
	configFile := c.String("config")
	if configFile == "" {
		home, err := utils.GetUserHome()
		if err != nil {
			return err
		}
		configFile = path.Join(home, defaultConfigFile)
	}
	if !utils.FileExists(configFile) {
		return fmt.Errorf("cannot find config file file: %s", configFile)
	}
	config, err := config2.NewConfigFromFile(configFile)
	if err != nil {
		return err
	}
	log := logger.InitLogger(config.LogLevel, config.LogPrettyPrint)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	server, err := proxy.NewServer(config)
	if err != nil {
		return err
	}

	metrics.Register()

	handler := proxy.PrepareRoutes(config, log, server)

	s := server.StartHTTPServer(handler)

	sig := <-stop
	log.Infof("Caught sig: %+v. Waiting process finishing...", sig)
	stopTimeout := 2
	ctxServer, doneServer := context.WithTimeout(context.Background(), time.Duration(stopTimeout)*time.Second)
	defer doneServer()
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
     params_in_cache_id:
       - 1
       - 2

   ---
   proxy_url: http://test.com
   port: 8080
   cache_methods:
   - name: method
     cache_by_params: true
     params_in_cache_name:
       - name1
       - name2
   `
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "config",
			Aliases:  []string{"c"},
			EnvVars:  []string{"RPC_PROXY_CONFIG_FILE"},
			Value:    "",
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
