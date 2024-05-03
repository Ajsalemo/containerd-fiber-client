package main

import (
	"log"

	controllers "containerd-custom-client/controllers"
	"github.com/gofiber/fiber/v2"
)

// Todo - replace loggers with something like `zapp`
// Todo - add route group
func main() {
	app := fiber.New()

	app.Get("/", controllers.Index)
	app.Get("/image/pull", controllers.PullImageController)
	app.Get("/image/list", controllers.ListImagesController)

    log.Fatal(app.Listen(":3000"))
}


