package database

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/charts"
	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/dto"
	"github.com/qiniu/qmgo"

	"go.mongodb.org/mongo-driver/bson"
)

type Posts struct {
	coll *qmgo.Collection
}

func NewPosts(coll *qmgo.Collection) *Posts {
	return &Posts{coll: coll}
}

func (p *Posts) Upsert(ctx context.Context, id int, post *dto.PostInfo) error {
	filter := bson.M{"_id": id}
	_ = p.coll.Remove(ctx, filter)
	doc := bson.M{"_id": id}
	postBytes, err := json.Marshal(post)
	if err != nil {
		return err
	}
	var postMap bson.M
	if err := json.Unmarshal(postBytes, &postMap); err != nil {
		return err
	}
	maps.Copy(doc, postMap)

	// Precompute chart stats
	if len(*post.Chart) > 0 {
		if chart, err := charts.UnmarshalSlice(*post.Chart); err == nil {
			stats := chart.Stats()
			doc["_chartStats"] = bson.M{
				"time":    stats.Time,
				"notes":   stats.Notes,
				"mainBPM": stats.MainBPM,
			}
		}
	}

	_, err = p.coll.InsertOne(ctx, doc)
	return err
}

func (p *Posts) GetByID(ctx context.Context, id int) (*dto.PostInfo, error) {
	var rawPost bson.M
	err := p.coll.Find(ctx, bson.M{"_id": id}).
		Select(bson.M{"_id": 0, "_chartStats": 0}).
		One(&rawPost)
	if err != nil {
		return nil, nil
	}
	var post dto.PostInfo
	postBytes, _ := json.Marshal(rawPost)
	if err := json.Unmarshal(postBytes, &post); err != nil {
		return nil, err
	}
	return &post, nil
}
func (p *Posts) Search(ctx context.Context, filter bson.M) ([]dto.PostInfo, error) {
	var rawPosts []bson.M
	err := p.coll.Find(ctx, filter).
		Select(bson.M{"_id": 0, "_chartStats": 0}).
		All(&rawPosts)
	if err != nil {
		return nil, err
	}
	posts := make([]dto.PostInfo, len(rawPosts))
	for i, rawPost := range rawPosts {
		postBytes, _ := json.Marshal(rawPost)
		if err := json.Unmarshal(postBytes, &posts[i]); err != nil {
			return nil, err
		}
	}
	return posts, nil
}
func (p *Posts) Delete(ctx context.Context, id int) error {
	return p.coll.Remove(ctx, bson.M{"_id": id})
}

// Database proxy methods for posts
func (d *Database) UpsertPost(ctx context.Context, id int, post *dto.PostInfo) error {
	return d.posts.Upsert(ctx, id, post)
}

func (d *Database) GetPostByID(ctx context.Context, id int) (*dto.PostInfo, error) {
	return d.posts.GetByID(ctx, id)
}

func (d *Database) SearchPosts(ctx context.Context, filter bson.M) ([]dto.PostInfo, error) {
	return d.posts.Search(ctx, filter)
}

func (d *Database) DeletePost(ctx context.Context, id int) error {
	return d.posts.Delete(ctx, id)
}

func (d *Database) GetNewestPostID(ctx context.Context) (int, error) {
	var maxDoc struct {
		ID int `bson:"_id"`
	}
	err := d.posts.coll.Find(ctx, bson.M{}).Sort("-_id").Limit(1).One(&maxDoc)
	return maxDoc.ID, err
}
