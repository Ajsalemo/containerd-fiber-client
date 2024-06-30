package controllers

import (
	ctr "containerd-custom-client/ctr"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func StopApplicationController(cxt *fiber.Ctx) error {
	err := ctr.StopTask()

	if err != nil {
		zap.L().Error("An error occurred when trying to stop the task..")
		zap.L().Error(err.Error())
		return cxt.Status(500).JSON(fiber.Map{"err": err.Error()})
	}

	return cxt.Status(200).JSON(fiber.Map{"msg": "Task stopped successfully"})
}
