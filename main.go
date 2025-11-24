package main

import (
	"anon-bestdori-database/app"
	"anon-bestdori-database/pkg/log"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"anon-bestdori-database/config"
)

var initDatabase = flag.Bool("init-database", false, "初始化数据库")

func main() {
	flag.Parse()

	conf, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v", err)
		os.Exit(1)
	}

	app.Run(conf, *initDatabase)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		select {
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				if err := conf.Reload(); err != nil {
					log.Errorf("failed to reload config: %v", err)
				} else {
					app.ReBoot(conf)
				}
			default:
				log.Info("application stopping...")
				app.Stop()
				return
			}
		case <-app.Stopped():
			app.Stop()
			return
		}
	}
}
