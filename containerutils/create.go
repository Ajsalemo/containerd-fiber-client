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
	defer container.Delete(ctxStdlib, containerd.WithSnapshotCleanup)

	zap.L().Info("Successfully created container with ID " + containerName + " and snapshot with ID " + containerName + "-snapshot")

	return err
}
