package controllers

import (
	containerd "github.com/containerd/containerd/v2/client"
	ctr "containerd-custom-client/ctr"

	"github.com/gofiber/fiber/v2"
	"log"
)

func PullImageController(cxt *fiber.Ctx) error {
	client, ctxStdlib, err := ctr.ContainerdClient()
	if err != nil {
		log.Println("An error occurred when using the containerd client..")
		log.Println(err)
		return err
	}

	// Close the client later on
	defer client.Close()

	image, err := client.Pull(ctxStdlib, "docker.io/library/redis:latest", containerd.WithPullUnpack)
	if err != nil {
		log.Println(err)
		return cxt.Status(500).JSON(fiber.Map{"err": err})
	}

	log.Printf("Successfully pulled %s image\n", image.Name())

	return cxt.JSON(fiber.Map{"msg": "Successfully pulled image " + image.Name()})
}
