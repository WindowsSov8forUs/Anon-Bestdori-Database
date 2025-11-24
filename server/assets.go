package server

import (
	"anon-bestdori-database/database"
	"anon-bestdori-database/files"

	"github.com/gofiber/fiber/v2"
)

func registerAssetsRoutes(group fiber.Router, _ *database.Database) {
	// /assets/musicjacket/{assetsName}
	group.Get("/musicjacket/:assetsName", func(c *fiber.Ctx) error {
		assetsName := c.Params("assetsName")
		fullPath := "musicjacket/" + assetsName

		data, err := files.GetAssets(fullPath)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// 假设 musicjacket 是图片
		c.Set("Content-Type", "image/png")
		c.Set("Cache-Control", "public, max-age=3600")
		return c.Send(data)
	})

	// /assets/sound/{assetsName}
	group.Get("/sound/:assetsName", func(c *fiber.Ctx) error {
		assetsName := c.Params("assetsName")
		fullPath := "sound/" + assetsName

		data, err := files.GetAssets(fullPath)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// 假设 sound 是音频文件
		c.Set("Content-Type", "audio/mp3")
		c.Set("Cache-Control", "public, max-age=3600")
		return c.Send(data)
	})
}
