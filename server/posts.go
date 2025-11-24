package server

import (
	"anon-bestdori-database/database"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

type PostsSearchParams struct {
	Keyword   string   `query:"keyword"`
	Artists   string   `query:"artists"`
	Diff      *int     `query:"diff"`
	Level     int      `query:"level"`
	LevelMin  int      `query:"level_min"`
	LevelMax  int      `query:"level_max"`
	Author    string   `query:"author"`
	Tags      []string `query:"tags"`
	Note      int      `query:"note"`
	NoteMin   int      `query:"note_min"`
	NoteMax   int      `query:"note_max"`
	BPM       float64  `query:"bpm"`
	BPMMin    float64  `query:"bpm_min"`
	BPMMax    float64  `query:"bpm_max"`
	Length    float64  `query:"length"`
	LengthMin float64  `query:"length_min"`
	LengthMax float64  `query:"length_max"`
}

func registerPostsRouter(router fiber.Router, db *database.Database) {
	router.Get("/search", getPostSearchHandler(db))
	router.Get("/:id", getPostIDHandler(db))
}

func getPostIDHandler(db *database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"result": false, "error": "无效的 ID 格式"})
		}

		post, err := db.GetPostByID(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"result": false, "error": err.Error()})
		}
		if post == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"result": false, "error": "帖子未找到"})
		}

		return c.JSON(fiber.Map{
			"result": true,
			"post":   post,
		})
	}
}

func getPostSearchHandler(db *database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		params := PostsSearchParams{}
		if err := c.QueryParser(&params); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"result": false, "error": "无效的查询参数"})
		}

		filters := []bson.M{}
		// keyword
		if params.Keyword != "" {
			filters = append(filters, bson.M{"title": bson.M{"$regex": params.Keyword, "$options": "i"}})
		}

		// artists
		if params.Artists != "" {
			filters = append(filters, bson.M{"artists": bson.M{"$regex": params.Artists, "$options": "i"}})
		}

		// diff
		if params.Diff != nil {
			filters = append(filters, bson.M{"diff": *params.Diff})
		}

		// level
		if params.Level != 0 || params.LevelMin != 0 || params.LevelMax != 0 {
			levelFilter := bson.M{}
			if params.Level != 0 {
				levelFilter["level"] = params.Level
			} else {
				rangeFilter := bson.M{}
				if params.LevelMin != 0 {
					rangeFilter["$gte"] = params.LevelMin
				}
				if params.LevelMax != 0 {
					rangeFilter["$lte"] = params.LevelMax
				}
				if len(rangeFilter) > 0 {
					levelFilter["level"] = rangeFilter
				}
			}
			if len(levelFilter) > 0 {
				filters = append(filters, levelFilter)
			}
		}

		// author
		if params.Author != "" {
			filters = append(filters, bson.M{"author.username": bson.M{"$regex": params.Author, "$options": "i"}})
		}

		// tags
		if len(params.Tags) > 0 {
			tagFilters := []bson.M{}
			for _, tag := range params.Tags {
				tagFilters = append(tagFilters, bson.M{
					"type": "text",
					"data": bson.M{
						"$regex":   "^" + tag + "$",
						"$options": "i",
					},
				})
			}
			filters = append(filters, bson.M{"tags": bson.M{"$all": tagFilters}})
		}

		// note
		if params.Note != 0 || params.NoteMin != 0 || params.NoteMax != 0 {
			noteFilter := bson.M{}
			if params.Note != 0 {
				noteFilter["_chartStats.notes"] = params.Note
			} else {
				rangeFilter := bson.M{}
				if params.NoteMin != 0 {
					rangeFilter["$gte"] = params.NoteMin
				}
				if params.NoteMax != 0 {
					rangeFilter["$lte"] = params.NoteMax
				}
				if len(rangeFilter) > 0 {
					noteFilter["_chartStats.notes"] = rangeFilter
				}
			}
			if len(noteFilter) > 0 {
				filters = append(filters, noteFilter)
			}
		}

		// bpm
		if params.BPM != 0 || params.BPMMin != 0 || params.BPMMax != 0 {
			bpmFilter := bson.M{}
			if params.BPM != 0 {
				bpmFilter["_chartStats.mainBPM"] = params.BPM
			} else {
				rangeFilter := bson.M{}
				if params.BPMMin != 0 {
					rangeFilter["$gte"] = params.BPMMin
				}
				if params.BPMMax != 0 {
					rangeFilter["$lte"] = params.BPMMax
				}
				if len(rangeFilter) > 0 {
					bpmFilter["_chartStats.mainBPM"] = rangeFilter
				}
			}
			if len(bpmFilter) > 0 {
				filters = append(filters, bpmFilter)
			}
		}

		// length
		if params.Length != 0 || params.LengthMin != 0 || params.LengthMax != 0 {
			lengthFilter := bson.M{}
			if params.Length != 0 {
				lengthFilter["_chartStats.time"] = params.Length
			} else {
				rangeFilter := bson.M{}
				if params.LengthMin != 0 {
					rangeFilter["$gte"] = params.LengthMin
				}
				if params.LengthMax != 0 {
					rangeFilter["$lte"] = params.LengthMax
				}
				if len(rangeFilter) > 0 {
					lengthFilter["_chartStats.time"] = rangeFilter
				}
			}
			if len(lengthFilter) > 0 {
				filters = append(filters, lengthFilter)
			}
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

		posts, err := db.SearchPosts(c.Context(), searchFilter)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"result": false, "error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"result": true,
			"count":  len(posts),
			"posts":  posts,
		})
	}
}
