package database

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/WindowsSov8forUs/bestdori-api-go/bestdori/dto"
	"github.com/qiniu/qmgo"

	"go.mongodb.org/mongo-driver/bson"
)

type Songs struct {
	coll *qmgo.Collection
}

func NewSongs(coll *qmgo.Collection) *Songs {
	return &Songs{coll: coll}
}

func (s *Songs) Upsert(ctx context.Context, id int, song *dto.SongInfo) error {
	filter := bson.M{"_id": id}
	_ = s.coll.Remove(ctx, filter)
	doc := bson.M{"_id": id}
	songBytes, err := json.Marshal(song)
	if err != nil {
		return err
	}
	var songMap bson.M
	if err := json.Unmarshal(songBytes, &songMap); err != nil {
		return err
	}
	maps.Copy(doc, songMap)

	// Precompute main BPM and store as internal field
	doc["_mainBPM"] = computeMainBPM(song.BPM)

	_, err = s.coll.InsertOne(ctx, doc)
	return err
}

func (s *Songs) GetByID(ctx context.Context, id int) (*dto.SongInfo, error) {
	var rawSong bson.M
	err := s.coll.Find(ctx, bson.M{"_id": id}).
		Select(bson.M{"_id": 0, "_mainBPM": 0}).
		One(&rawSong)
	if err != nil {
		return nil, nil
	}
	var song dto.SongInfo
	songBytes, _ := json.Marshal(rawSong)
	if err := json.Unmarshal(songBytes, &song); err != nil {
		return nil, err
	}
	return &song, nil
}
func (s *Songs) Search(ctx context.Context, filter bson.M) ([]dto.SongInfo, error) {
	var rawSongs []bson.M
	err := s.coll.Find(ctx, filter).
		Select(bson.M{"_id": 0, "_mainBPM": 0}).
		All(&rawSongs)
	if err != nil {
		return nil, err
	}
	songs := make([]dto.SongInfo, len(rawSongs))
	for i, rawSong := range rawSongs {
		songBytes, _ := json.Marshal(rawSong)
		if err := json.Unmarshal(songBytes, &songs[i]); err != nil {
			return nil, err
		}
	}
	return songs, nil
}
func (s *Songs) Delete(ctx context.Context, id int) error {
	return s.coll.Remove(ctx, bson.M{"_id": id})
}

func computeMainBPM(bpmMap map[string][]dto.SongBPM) float64 {
	if bpmMap == nil {
		return 0
	}
	// prefer "3" field if available
	segs, ok := bpmMap["3"]
	if !ok || len(segs) == 0 {
		// fallback: pick any key with data
		for _, v := range bpmMap {
			if len(v) > 0 {
				segs = v
				break
			}
		}
	}
	if len(segs) == 0 {
		return 0
	}
	durByBPM := make(map[float64]float64)
	for _, seg := range segs {
		dur := seg.End - seg.Start
		if dur > 0 {
			durByBPM[seg.BPM] += dur
		}
	}
	if len(durByBPM) == 0 {
		return 0
	}
	var main float64
	maxDur := float64(-1)
	for b, d := range durByBPM {
		if d > maxDur {
			maxDur = d
			main = b
		}
	}
	return main
}

// Database proxy methods for songs
func (d *Database) UpsertSong(ctx context.Context, id int, song *dto.SongInfo) error {
	return d.songs.Upsert(ctx, id, song)
}

func (d *Database) GetSongByID(ctx context.Context, id int) (*dto.SongInfo, error) {
	return d.songs.GetByID(ctx, id)
}

func (d *Database) SearchSongs(ctx context.Context, filter bson.M) ([]dto.SongInfo, error) {
	return d.songs.Search(ctx, filter)
}

func (d *Database) DeleteSong(ctx context.Context, id int) error {
	return d.songs.Delete(ctx, id)
}
