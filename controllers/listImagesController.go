package controllers

import (
	ctr "containerd-custom-client/ctr"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func ListImagesController(cxt *fiber.Ctx) error {
	images, err := ctr.ListImages()
	if err != nil {
		zap.L().Error(err.Error())
		return cxt.Status(500).JSON(fiber.Map{"err": err})
	}

	return cxt.JSON(fiber.Map{"msg": images})
}
