package controllers

import (
	ctr "containerd-custom-client/ctr"

	"github.com/gofiber/fiber/v2"
	"log"
)

func ListImagesController(cxt *fiber.Ctx) error {
	type Image struct {
		Name string
	}
	var imageArray []Image

	client, ctxStdlib, err := ctr.ContainerdClient()
	if err != nil {
		log.Println("An error occurred when using the containerd client..")
		log.Println(err)
		return err
	}

	// Close the client later on
	defer client.Close()

	images, err := client.ListImages(ctxStdlib)
	if err != nil {
		log.Println(err)
		return cxt.Status(500).JSON(fiber.Map{"err": err})
	}

	for _, image := range images {
		log.Println(" - " + image.Name())
		imageArray = append(imageArray, Image{
			Name: image.Name(),
		})
	}

	return cxt.JSON(fiber.Map{"msg": imageArray})
}
