package app

import (
	"anon-bestdori-database/config"
	"anon-bestdori-database/data"
	"anon-bestdori-database/database"
	"anon-bestdori-database/pkg/log"
	"anon-bestdori-database/server"
	"anon-bestdori-database/version"
	"context"
	"fmt"
	"net"
	"strings"
)

var stoppedChan = make(chan bool, 1)

func Stopped() chan bool {
	return stoppedChan
}

type app struct {
	ctx     context.Context
	cancel  context.CancelFunc
	conf    *config.Config
	db      *database.Database
	updater *data.DataUpdater
	server  *server.Server
}

var appInstance *app

func newApp(conf *config.Config) (*app, error) {
	ctx, cancel := context.WithCancel(context.Background())

	db, err := database.NewClient(ctx, conf)
	if err != nil {
		cancel()
		log.Errorf("failed to connect to database: %v", err)
		return nil, err
	}
	log.Infof("connection with database established: %s", conf.Mongo.URI)

	updater := data.NewDataUpdater(db, conf, ctx)
	srv := server.New(db)

	return &app{
		ctx:     ctx,
		cancel:  cancel,
		conf:    conf,
		db:      db,
		updater: updater,
		server:  srv,
	}, nil
}

func (a *app) Init() error {
	log.Info("starting application data initialization...")
	if err := a.updater.Init(); err != nil {
		log.Errorf("application data initialization failed: %v", err)
		return err
	}
	log.Info("application data initialization completed")
	return nil
}

func (a *app) Run() {
	a.updater.StartUpdating()
	log.Info("scheduled update started")

	addr := net.JoinHostPort(a.conf.Server.Host, a.conf.Server.Port)
	log.Infof("application listening on %s", addr)
	go func() {
		if err := a.server.Start(a.ctx, addr); err != nil && !strings.Contains(err.Error(), "server closed") {
			log.Errorf("failed to run fiber server: %v", err)
			stoppedChan <- true
			return
		}
		log.Info("fiber server stopped")
	}()
}

func (a *app) Close() {
	log.Info("closing connection with database...")
	a.db.Close(a.ctx)
	a.cancel()
	log.Info("application stopped")
}

func Run(conf *config.Config, init bool) error {
	log.Init(conf, "anon-bestdori-database")

	if appInstance != nil {
		return fmt.Errorf("application is running")
	}

	app, err := newApp(conf)
	if err != nil {
		return err
	}
	appInstance = app

	if init {
		err = app.Init()
		if err != nil {
			app.Close()
			appInstance = nil
			return err
		}
	}

	app.Run()
	log.Infof("application version: %s", version.Version)
	return nil
}

func Stop() {
	if appInstance == nil {
		log.Error("no application running")
	}

	appInstance.Close()
	appInstance = nil
}

func ReBoot(conf *config.Config) error {
	log.Info("application rebooting...")
	Stop()
	return Run(conf, false)
}
