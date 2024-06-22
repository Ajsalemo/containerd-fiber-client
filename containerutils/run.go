package containerutils

import (
	ctr "containerd-custom-client/ctr"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"go.uber.org/zap"
)

func RunTask(container containerd.Container, containerName string) error {
	client, ctxStdlib, err := ctr.ContainerdClient()
	if err != nil {
		zap.L().Error("An error occurred when using the containerd client..")
		return err
	}

	// Close the client later on
	defer client.Close()
	// Create a new task with the container passed in as a parameter
	// Note, this is not an actual running container. We need to create a 'task' to run the container
	task, err := container.NewTask(ctxStdlib, cio.NewCreator(cio.WithStdio))
	if err != nil {
		zap.L().Error("An error occurred when trying to create a task from container: " + containerName)
		return err
	}
	defer task.Delete(ctxStdlib)
	// Run this as a goroutine to avoid blocking
	// This runs it in the background - but we can still see the container/task stdout/stderr in our Go process
	go func() error {
		// See https://github.com/containerd/containerd/blob/main/docs/getting-started.md
		// We use `Wait` to avoid issues with processes that exit/complete very quick
		exitStatusC, err := task.Wait(ctxStdlib)
		if err != nil {
			zap.L().Error("An error occurred when trying to use `task.Wait` on task: " + task.ID())
			return err
		}

		if err := task.Start(ctxStdlib); err != nil {
			zap.L().Error("An error occurred when trying to start a task with: " + task.ID())
		}
		status := <-exitStatusC
		code, _, err := status.Result()
		if err != nil {
			return err
		}
		zap.L().Info("Task " + task.ID() + " exited with status code: " + string(code))

		return err
	}()

	// Task succesfully created
	zap.L().Info("Successfully created task with ID " + task.ID())

	return err
}
