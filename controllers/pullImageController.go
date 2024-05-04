package controllers

import (
	ctr "containerd-custom-client/ctr"
	containerd "github.com/containerd/containerd/v2/client"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func PullImageController(cxt *fiber.Ctx) error {
	client, ctxStdlib, err := ctr.ContainerdClient()
	if err != nil {
		zap.L().Error("An error occurred when using the containerd client..")
		zap.L().Fatal(err.Error())
	}

	// Close the client later on
	defer client.Close()

	image, err := client.Pull(ctxStdlib, "docker.io/library/redis:latest", containerd.WithPullUnpack)
	if err != nil {
		zap.L().Error(err.Error())
		return cxt.Status(500).JSON(fiber.Map{"err": err})
	}

	zap.L().Info("Successfully pulled image" + image.Name())

	return cxt.JSON(fiber.Map{"msg": "Successfully pulled image " + image.Name()})
}
