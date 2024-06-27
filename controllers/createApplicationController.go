package controllers

import (
	ctr "containerd-custom-client/ctr"
	"encoding/json"

	"github.com/containerd/containerd/v2/core/remotes/docker"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func CreateApplicationController(cxt *fiber.Ctx) error {
	var imageDefinition ctr.ImageDefintion
	var ctrImageProps ctr.CtrImageProps

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

		err := ctr.PullAuthenticatedImage(imageDefinition, resolver)

		if err != nil {
			zap.L().Error("An error occured when trying to pull an authenticated image: " + imageDefinition.Registry + "/" + imageDefinition.Image + ":" + imageDefinition.Tag)
			// zap.L().Error(err.Error())
			return cxt.Status(500).JSON(fiber.Map{"err": err.Error()})
		}

		zap.L().Info("Successfully pulled image " + ctrImageProps.PulledImage)
		// Create the container after a successful pull
		// CreateContainer calls RunTask which is what actually starts the application procoess
		cerr := ctr.CreateContainer(imageDefinition)

		if cerr != nil {
			zap.L().Error("An error occurred when trying to create a container for authenticated image: " + ctrImageProps.CtrContainerDef.ID() + " and image " + ctrImageProps.PulledImage)
			zap.L().Error(cerr.Error())
			return cxt.Status(500).JSON(fiber.Map{"err": cerr.Error()})
		}
	} else {
		// Else, pull without the resolver since it's public
		err := ctr.PullPublicImage(imageDefinition)
		if err != nil {
			zap.L().Error("An error occured when trying to pull a public image: " + imageDefinition.Registry + "/" + imageDefinition.Image + ":" + imageDefinition.Tag)
			zap.L().Error(err.Error())
			return cxt.Status(500).JSON(fiber.Map{"err": err.Error()})
		}

		zap.L().Info("Successfully pulled image " + ctrImageProps.PulledImage)

		// Create the container after a successful pull
		// CreateContainer calls RunTask which is what actually starts the application procoess
		cerr := ctr.CreateContainer(imageDefinition)

		if cerr != nil {
			zap.L().Error("An error occurred when trying to create a container for authenticated image: " + ctrImageProps.CtrContainerDef.ID() + " and image " + ctrImageProps.PulledImage)
			zap.L().Error(cerr.Error())
			return cxt.Status(500).JSON(fiber.Map{"err": cerr.Error()})
		}
	}

	return cxt.JSON(fiber.Map{"msg": "Successfully pulled image " + ctrImageProps.PulledImage})
}
