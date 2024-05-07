package controllers

import (
	ctr "containerd-custom-client/ctr"
	"encoding/json"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type ImageDefintion struct {
	Registry  string `json:"registry"`
	Image string `json:"image"`
	Tag string `json:"tag"`
}

func PullImageController(cxt *fiber.Ctx) error {
	var imageDefinition ImageDefintion
	body := cxt.Body()

	if body == nil {
		zap.L().Error("Request body is nil")
		return cxt.Status(400).JSON(fiber.Map{"msg": "Request body is empty"})
	}	

	// Unmarshal the JSON into a ImageDefinition object
	// Return an error if the JSON is invalid
	err := json.Unmarshal(body, &imageDefinition)
	if err != nil {
		zap.L().Error(err.Error())
		return cxt.Status(500).JSON(fiber.Map{"err": err.Error()})
	}

	fmt.Println(imageDefinition.Image + "\n")
	fmt.Println(imageDefinition.Image + "\n")
	fmt.Println(imageDefinition.Image)

	client, ctxStdlib, err := ctr.ContainerdClient()
	if err != nil {
		zap.L().Error("An error occurred when using the containerd client..")
		return err
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
