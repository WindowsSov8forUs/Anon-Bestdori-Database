package database

import (
	"context"

	"github.com/qiniu/qmgo"

	"anon-bestdori-database/config"
)

type Database struct {
	cli    *qmgo.Client
	posts  *Posts
	songs  *Songs
	charts *Charts
}

func NewClient(ctx context.Context, conf *config.Config) (*Database, error) {
	cli, err := qmgo.NewClient(ctx, &qmgo.Config{Uri: conf.Mongo.URI})
	if err != nil {
		return nil, err
	}
	if err = cli.Ping(10); err != nil {
		cli.Close(ctx)
		return nil, err
	}

	db := cli.Database("anon_db")

	return &Database{
		cli:    cli,
		posts:  NewPosts(db.Collection("posts")),
		songs:  NewSongs(db.Collection("songs")),
		charts: NewCharts(db.Collection("charts")),
	}, nil
}

func (d *Database) Close(ctx context.Context) error {
	return d.cli.Close(ctx)
}
