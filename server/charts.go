package server

import (
	"anon-bestdori-database/database"

	"github.com/gofiber/fiber/v2"
)

func registerChartsRoutes(router fiber.Router, db *database.Database) {
	router.Get("/:songId/:diff", getChartsIdDiffHandler(db))
}

func getChartsIdDiffHandler(db *database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		songId := c.Params("songId")
		diff := c.Params("diff")
		id := songId + "-" + diff

		chart, err := db.GetChartByID(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"result": false, "error": err.Error()})
		}
		if chart == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"result": false, "error": "谱面未找到"})
		}

		return c.JSON(fiber.Map{
			"result": true,
			"chart":  chart,
		})
	}
}
