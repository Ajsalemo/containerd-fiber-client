package controllers

import (
	ctr "containerd-custom-client/ctr"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func ListImagesController(cxt *fiber.Ctx) error {
	type Image struct {
		Name string
	}
	var imageArray []Image

	client, ctxStdlib, err := ctr.ContainerdClient()
	if err != nil {
		zap.L().Error("An error occurred when using the containerd client..")
		zap.L().Fatal(err.Error())
	}

	// Close the client later on
	defer client.Close()

	images, err := client.ListImages(ctxStdlib)
	if err != nil {
		zap.L().Error(err.Error())
		return cxt.Status(500).JSON(fiber.Map{"err": err})
	}

	for _, image := range images {
		zap.L().Info(image.Name())
		imageArray = append(imageArray, Image{
			Name: image.Name(),
		})
	}

	return cxt.JSON(fiber.Map{"msg": imageArray})
}
