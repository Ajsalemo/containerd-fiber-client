package main

import (
	controllers "containerd-custom-client/controllers"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
}

func main() {
	app := fiber.New()
	api := app.Group("/api/images")

	app.Get("/", controllers.Index)
	api.Post("/pull", controllers.PullImageController)
	api.Get("/list", controllers.ListImagesController)

	err := app.Listen(":3000")

	if err != nil {
		zap.L().Fatal(err.Error())
	}
}
