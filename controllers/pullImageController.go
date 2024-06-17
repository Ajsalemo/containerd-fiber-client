package controllers

import (
	containerutils "containerd-custom-client/containerutils"
	ctr "containerd-custom-client/ctr"
	"encoding/json"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/remotes/docker"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type ImageDefintion struct {
	Registry         string `json:"registry"`
	Image            string `json:"image"`
	Tag              string `json:"tag"`
	IsPublic         *bool  `json:"isPublic"`
	RegistryUsername string `json:"registryUsername"`
	RegistryPassword string `json:"registryPassword"`
	ContainerName    string `json:"containerName"`
	// This field will be populated with the name of the image that was pulled - this is NOT on the incoming request body from the client
	// All the other fields above are
	PulledImage string `json:"pulledImage"`
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

	// Do some validation checks on inputs
	if imageDefinition.Registry == "" {
		return cxt.Status(400).JSON(fiber.Map{"err": "Registry name is empty or was not provided in the request body"})
	} else if imageDefinition.Image == "" {
		return cxt.Status(400).JSON(fiber.Map{"err": "Image name is empty or was not provided in the request body"})
	} else if imageDefinition.Tag == "" {
		return cxt.Status(400).JSON(fiber.Map{"err": "Tag name is empty or was not provided in the request body"})
	} else if imageDefinition.RegistryUsername == "" {
		return cxt.Status(400).JSON(fiber.Map{"err": "Registry username is empty or was not provided in the request body"})
	} else if imageDefinition.RegistryPassword == "" {
		return cxt.Status(400).JSON(fiber.Map{"err": "Registry password is empty or was not provided in the request body"})
	} else if imageDefinition.IsPublic == nil {
		return cxt.Status(400).JSON(fiber.Map{"err": "IsPublic field is empty or was not provided in the request body"})
	} else if imageDefinition.ContainerName == "" {
		return cxt.Status(400).JSON(fiber.Map{"err": "Container name is empty or was not provided in the request body"})
	}

	// Check if the pull is public or not - dictated by the `IsPublic` field
	// If it's not public, we need to provide credentials
	// Note: We're dereferencing the pointer to get the actual value with '*'
	// IsPublic is set to an uptyped bool in the ImageDefinition struct to check if it's nil on the incoming Request Body
	// We need to derefernce or else we get a mismatched type error (untyped bool vs bool)
	if !*imageDefinition.IsPublic {
		resolver := docker.NewResolver(docker.ResolverOptions{
			Credentials: func(host string) (string, string, error) {
				return imageDefinition.RegistryUsername, imageDefinition.RegistryPassword, nil
			},
		})

		image, err := client.Pull(ctxStdlib, imageDefinition.Registry+"/"+imageDefinition.Image+":"+imageDefinition.Tag, containerd.WithPullUnpack, containerd.WithResolver(resolver))
		if err != nil {
			zap.L().Error(err.Error())
			return cxt.Status(500).JSON(fiber.Map{"err": err.Error()})
		}

		zap.L().Info("Successfully pulled image " + image.Name())
		imageDefinition.PulledImage = image.Name()
		// Create the container after a successful pull
		err2 := containerutils.CreateContainer(image, imageDefinition.ContainerName)

		if err2 != nil {
			zap.L().Error(err2.Error())
			return cxt.Status(500).JSON(fiber.Map{"err": err2.Error()})
		}
	} else {
		// Else, pull without the resolver since it's public
		image, err := client.Pull(ctxStdlib, imageDefinition.Registry+"/"+imageDefinition.Image+":"+imageDefinition.Tag, containerd.WithPullUnpack)
		if err != nil {
			zap.L().Error(err.Error())
			return cxt.Status(500).JSON(fiber.Map{"err": err.Error()})
		}

		zap.L().Info("Successfully pulled image " + image.Name())
		imageDefinition.PulledImage = image.Name()

		imageDefinition.PulledImage = image.Name()
		// Create the container after a successful pull
		err2 := containerutils.CreateContainer(image, imageDefinition.ContainerName)

		if err2 != nil {
			zap.L().Error(err2.Error())
			return cxt.Status(500).JSON(fiber.Map{"err": err2.Error()})
		}
	}

	return cxt.JSON(fiber.Map{"msg": "Successfully pulled image " + imageDefinition.PulledImage})
}
