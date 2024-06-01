package controllers

import (
	ctr "containerd-custom-client/ctr"
	"encoding/json"

	containerd "github.com/containerd/containerd/v2/client"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type ImageDefintion struct {
	Registry string `json:"registry"`
	Image    string `json:"image"`
	Tag      string `json:"tag"`
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

	client, ctxStdlib, err := ctr.ContainerdClient()
	if err != nil {
		zap.L().Error("An error occurred when using the containerd client..")
		return err
	}

	// Close the client later on
	defer client.Close()
	// Small helper to change docker.io to what containerd is expecting
	if imageDefinition.Registry == "docker.io" {
		imageDefinition.Registry = "docker.io/library"
	}
	// Do some validation checks on inputs
	if imageDefinition.Registry == "" {
		return cxt.Status(400).JSON(fiber.Map{"err": "Registry name is empty or was not provided in the request body"})
	} else if imageDefinition.Image == "" {
		return cxt.Status(400).JSON(fiber.Map{"err": "Image name is empty or was not provided in the request body"})
	} else if imageDefinition.Tag == "" {
		return cxt.Status(400).JSON(fiber.Map{"err": "Tag name is empty or was not provided in the request body"})
	}

	image, err := client.Pull(ctxStdlib, imageDefinition.Registry+"/"+imageDefinition.Image+":"+imageDefinition.Tag, containerd.WithPullUnpack)
	if err != nil {
		zap.L().Error(err.Error())
		return cxt.Status(500).JSON(fiber.Map{"err": err.Error()})
	}

	zap.L().Info("Successfully pulled image " + image.Name())

	return cxt.JSON(fiber.Map{"msg": "Successfully pulled image " + image.Name()})
}
