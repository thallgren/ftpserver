// ftpserver allows to create your own FTP(S) server
package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	gklog "github.com/go-kit/kit/log"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/fclairamb/ftpserverlib/log"

	"github.com/fclairamb/ftpserver/config"
	"github.com/fclairamb/ftpserver/server"
)

var (
	ftpServer *ftpserver.FtpServer
)

func main() {
	// Arguments vars
	var confFile string
	var onlyConf bool

	// Parsing arguments
	flag.StringVar(&confFile, "conf", "", "Configuration file")
	flag.BoolVar(&onlyConf, "conf-only", false, "Only create the conf")
	flag.Parse()

	// Setting up the logger
	logger := log.NewGKLogger(gklog.NewLogfmtLogger(gklog.NewSyncWriter(os.Stdout))).With(
		"ts", gklog.DefaultTimestampUTC,
		"caller", gklog.DefaultCaller,
	)

	autoCreate := onlyConf

	// The general idea here is that if you start it without any arg, you're probably doing a local quick&dirty run
	// possibly on a windows machine, so we're better of just using a default file name and create the file.
	if confFile == "" {
		confFile = "ftpserver.json"
		autoCreate = true
	}

	if autoCreate {
		if _, err := os.Stat(confFile); err != nil && os.IsNotExist(err) {
			logger.Warn("No conf file, creating one", "confFile", confFile)

			if err := ioutil.WriteFile(confFile, confFileContent(), 0644); err != nil {
				logger.Warn("Couldn't create conf file", "confFile", confFile)
			}
		}
	}

	conf, errConfig := config.NewConfig(confFile, logger)
	if errConfig != nil {
		logger.Error("Can't load conf", errConfig)
		return
	}

	// Loading the driver
	driver, err := server.NewServer(conf, logger.With("component", "driver"))

	if err != nil {
		logger.Error("Could not load the driver", err)
		return
	}

	// Instantiating the server by passing our driver implementation
	ftpServer = ftpserver.NewFtpServer(driver)

	// Overriding the server default silent logger by a sub-logger (component: server)
	ftpServer.Logger = logger.With("component", "server")

	// Preparing the SIGTERM handling
	go signalHandler()

	// Blocking call, behaving similarly to the http.ListenAndServe
	if onlyConf {
		logger.Warn("Only creating conf")
		return
	}

	if err := ftpServer.ListenAndServe(); err != nil {
		logger.Error("Problem listening", err)
	}
}

func signalHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)

	for {
		sig := <-ch

		if sig == syscall.SIGTERM {
			ftpServer.Stop()
		}
	}
}

func confFileContent() []byte {
	str := `{
  "version": 1,
  "accesses": [
    {
      "user": "test",
      "pass": "test",
      "fs": "os",
      "params": {
        "basePath": "/tmp"
      }
    }
  ],
  "passive_transfer_port_range": {
    "start": 2122,
    "end": 2130
  }
}`

	return []byte(str)
}
