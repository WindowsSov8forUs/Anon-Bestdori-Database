package database

import (
	"context"

	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/dto"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
)

type Charts struct {
	coll *qmgo.Collection
}

func NewCharts(coll *qmgo.Collection) *Charts {
	return &Charts{coll: coll}
}

func (c *Charts) Upsert(ctx context.Context, id string, chart *dto.Chart) error {
	filter := bson.M{"_id": id}
	_ = c.coll.Remove(ctx, filter)
	doc := bson.M{
		"_id":   id,
		"chart": chart,
	}
	_, err := c.coll.InsertOne(ctx, doc)
	return err
}

func (c *Charts) GetByID(ctx context.Context, id string) (*dto.Chart, error) {
	type result struct {
		Chart dto.Chart `bson:"chart"`
	}
	var res result
	err := c.coll.Find(ctx, bson.M{"_id": id}).One(&res)
	if err != nil {
		return nil, nil
	}
	return &res.Chart, nil
}

// Database proxy methods for charts
func (d *Database) UpsertChart(ctx context.Context, id string, chart *dto.Chart) error {
	return d.charts.Upsert(ctx, id, chart)
}

func (d *Database) GetChartByID(ctx context.Context, id string) (*dto.Chart, error) {
	return d.charts.GetByID(ctx, id)
}
