package server

import (
	"fmt"
	"strconv"

	"anon-bestdori-database/database"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

type SongsSearchParams struct {
	Keyword   string  `query:"keyword"`
	Diff      *int    `query:"diff"`
	Level     int     `query:"level"`
	LevelMin  int     `query:"level_min"`
	LevelMax  int     `query:"level_max"`
	Notes     int     `query:"notes"`
	NotesMin  int     `query:"notes_min"`
	NotesMax  int     `query:"notes_max"`
	BPM       float64 `query:"bpm"`
	BPMMin    float64 `query:"bpm_min"`
	BPMMax    float64 `query:"bpm_max"`
	Length    float64 `query:"length"`
	LengthMin float64 `query:"length_min"`
	LengthMax float64 `query:"length_max"`
	BandId    int     `query:"bandId"`
	Tag       string  `query:"tag"`
}

func registerSongsRoutes(router fiber.Router, db *database.Database) {
	router.Get("/search", getSongsSearchHandler(db))
	router.Get("/:id", getSongsIDHandler(db))
}

func getSongsIDHandler(db *database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"result": false, "error": "无效的 ID 格式"})
		}

		song, err := db.GetSongByID(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"result": false, "error": err.Error()})
		}
		if song == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"result": false, "error": "歌曲未找到"})
		}

		return c.JSON(fiber.Map{
			"result": true,
			"song":   song,
		})
	}
}

func getSongsSearchHandler(db *database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		params := SongsSearchParams{}
		if err := c.QueryParser(&params); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"result": false, "error": "无效的查询参数"})
		}

		filters := []bson.M{}
		// keyword
		if params.Keyword != "" {
			titleFilter := bson.M{
				"$elemMatch": bson.M{
					"$ne":      nil,
					"$regex":   params.Keyword,
					"$options": "i",
				},
			}
			filters = append(filters, bson.M{"musicTitle": titleFilter})
		}

		// diff
		if params.Diff != nil {
			diffKey := fmt.Sprintf("difficulty.%d", *params.Diff)
			filters = append(filters, bson.M{diffKey: bson.M{"$exists": true}})
		}

		// level
		if params.Level != 0 || params.LevelMin != 0 || params.LevelMax != 0 {
			levelFilter := bson.M{}
			if params.Level != 0 {
				if params.Diff != nil {
					key := fmt.Sprintf("difficulty.%d.playLevel", *params.Diff)
					levelFilter = bson.M{key: params.Level}
				} else {
					levelFilter["$or"] = []bson.M{}
					for i := range 5 {
						key := fmt.Sprintf("difficulty.%d.playLevel", i)
						levelFilter["$or"] = append(levelFilter["$or"].([]bson.M), bson.M{key: params.Level})
					}
				}
			} else if params.LevelMin != 0 || params.LevelMax != 0 {
				rangeFilter := bson.M{}
				if params.LevelMin != 0 {
					rangeFilter["$gte"] = params.LevelMin
				}
				if params.LevelMax != 0 {
					rangeFilter["$lte"] = params.LevelMax
				}
				if params.Diff != nil {
					key := fmt.Sprintf("difficulty.%d.playLevel", *params.Diff)
					levelFilter = bson.M{key: rangeFilter}
				} else {
					levelFilter["$or"] = []bson.M{}
					for i := range 5 {
						key := fmt.Sprintf("difficulty.%d.playLevel", i)
						levelFilter["$or"] = append(levelFilter["$or"].([]bson.M), bson.M{key: rangeFilter})
					}
				}
			}
			filters = append(filters, levelFilter)
		}

		// notes
		if params.Notes != 0 || params.NotesMin != 0 || params.NotesMax != 0 {
			notesFilter := bson.M{}
			if params.Notes != 0 {
				if params.Diff != nil {
					key := fmt.Sprintf("notes.%d", *params.Diff)
					notesFilter = bson.M{key: params.Notes}
				} else {
					notesFilter["$or"] = []bson.M{}
					for i := range 5 {
						key := fmt.Sprintf("notes.%d", i)
						notesFilter["$or"] = append(notesFilter["$or"].([]bson.M), bson.M{key: params.Notes})
					}
				}
			} else if params.NotesMin != 0 || params.NotesMax != 0 {
				rangeFilter := bson.M{}
				if params.NotesMin != 0 {
					rangeFilter["$gte"] = params.NotesMin
				}
				if params.NotesMax != 0 {
					rangeFilter["$lte"] = params.NotesMax
				}
				if params.Diff != nil {
					key := fmt.Sprintf("notes.%d", *params.Diff)
					notesFilter = bson.M{key: rangeFilter}
				} else {
					notesFilter["$or"] = []bson.M{}
					for i := range 5 {
						key := fmt.Sprintf("notes.%d", i)
						notesFilter["$or"] = append(notesFilter["$or"].([]bson.M), bson.M{key: rangeFilter})
					}
				}
			}
			filters = append(filters, notesFilter)
		}

		// bpm
		if params.BPM != 0 || params.BPMMin != 0 || params.BPMMax != 0 {
			bpmFilter := bson.M{}
			if params.BPM != 0 {
				bpmFilter = bson.M{"_mainBPM": params.BPM}
			} else if params.BPMMin != 0 || params.BPMMax != 0 {
				rangeFilter := bson.M{}
				if params.BPMMin != 0 {
					rangeFilter["$gte"] = params.BPMMin
				}
				if params.BPMMax != 0 {
					rangeFilter["$lte"] = params.BPMMax
				}
				bpmFilter = bson.M{"_mainBPM": rangeFilter}
			}
			filters = append(filters, bpmFilter)
		}

		// length
		if params.Length != 0 || params.LengthMin != 0 || params.LengthMax != 0 {
			lengthFilter := bson.M{}
			if params.Length != 0 {
				lengthFilter = bson.M{"length": params.Length}
			} else if params.LengthMin != 0 || params.LengthMax != 0 {
				rangeFilter := bson.M{}
				if params.LengthMin != 0 {
					rangeFilter["$gte"] = params.LengthMin
				}
				if params.LengthMax != 0 {
					rangeFilter["$lte"] = params.LengthMax
				}
				lengthFilter = bson.M{"length": rangeFilter}
			}
			filters = append(filters, lengthFilter)
		}

		// bandId
		if params.BandId != 0 {
			bandFilter := bson.M{"bandId": params.BandId}
			filters = append(filters, bandFilter)
		}

		// tag
		if params.Tag != "" {
			tagFilter := bson.M{"tag": params.Tag}
			filters = append(filters, tagFilter)
		}

		// build final filter
		var searchFilter bson.M
		if len(filters) == 0 {
			searchFilter = bson.M{}
		} else if len(filters) == 1 {
			searchFilter = filters[0]
		} else {
			searchFilter = bson.M{"$and": filters}
		}

		songs, err := db.SearchSongs(c.Context(), searchFilter)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"result": false, "error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"result": true,
			"count":  len(songs),
			"songs":  songs,
		})
	}
}
