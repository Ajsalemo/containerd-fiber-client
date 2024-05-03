package controllers

import (
	"github.com/gofiber/fiber/v2"
)

func Index(cxt *fiber.Ctx) error {
	return cxt.JSON(fiber.Map{"msg": "containerd-custom-client"})
}