package containerutils

import (
	ctr "containerd-custom-client/ctr"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/oci"
	"go.uber.org/zap"
)

func CreateContainer(image containerd.Image, containerName string) error {
	client, ctxStdlib, err := ctr.ContainerdClient()
	if err != nil {
		zap.L().Error("An error occurred when using the containerd client..")
		return err
	}

	// Close the client later on
	defer client.Close()
	// Create a new container with the image
	// Note, this is not an actual running container. We need to create a 'task' to run the container
	container, err := client.NewContainer(
		ctxStdlib,
		containerName,
		containerd.WithNewSnapshot(containerName+"-snapshot", image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)

	if err != nil {
		zap.L().Error("An error occurred when trying to create a new container for image: " + image.Name())
		return err
	}
	zap.L().Info("Successfully created container with ID " + containerName + " and snapshot with ID " + containerName + "-snapshot")
	zap.L().Info("Attempting to create a new task for container: " + containerName)
	// Start a task (process)
	taskErr := RunTask(container, containerName)

	if taskErr != nil {
		zap.L().Error("An error occurred when trying to create a new task for container: " + containerName)
		return err
	}

	defer container.Delete(ctxStdlib, containerd.WithSnapshotCleanup)

	return err
}
