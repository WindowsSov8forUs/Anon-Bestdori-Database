/*
   Copyright (C) 2025 WindowsSov8forUs

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published
   by the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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
