package ctr

import (
	"context"

	"go.uber.org/zap"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/remotes"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/oci"
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

type Image struct {
	Name string
}

var imageDefinition ImageDefintion

func ContainerdClient() (*containerd.Client, context.Context, error) {
	client, err := containerd.New("/run/containerd/containerd.sock")
	ctx := namespaces.WithNamespace(context.Background(), "default")

	if err != nil {
		zap.L().Error("An error occurred when using the containerd client..")
		return client, ctx, err
	}

	// Close the client
	defer client.Close()

	return client, ctx, err
}

// Function to pull authenticated images
// The primary difference here is the use of `Resolver` to handle authentication
func PullAuthenticatedImage(resolver remotes.Resolver) (containerd.Image, error) {
	client, ctx, _ := ContainerdClient()
	image, err := client.Pull(ctx, imageDefinition.Registry+"/"+imageDefinition.Image+":"+imageDefinition.Tag, containerd.WithPullUnpack, containerd.WithResolver(resolver))

	if err != nil {
		zap.L().Error("An error occurred when trying to pull an image..")
		return image, err
	}
	return image, err
}

// Function to pull public images
// No authentication
func PullPublicImage() (containerd.Image, error) {
	client, ctx, _ := ContainerdClient()
	image, err := client.Pull(ctx, imageDefinition.Registry+"/"+imageDefinition.Image+":"+imageDefinition.Tag, containerd.WithPullUnpack)

	if err != nil {
		zap.L().Error("An error occurred when trying to pull an image..")
		return image, err
	}
	return image, err
}

func CreateContainer(image containerd.Image) (containerd.Container, error) {
	client, ctx, _ := ContainerdClient()
	// Create a new container with the image
	// Note, this is not an actual running container. We need to create a 'task' to run the container
	container, err := client.NewContainer(
		ctx,
		imageDefinition.ContainerName,
		containerd.WithNewSnapshot(imageDefinition.ContainerName+"-snapshot", image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)

	if err != nil {
		zap.L().Error("An error occurred when trying to create a new container for image: " + image.Name())
		return container, err
	}
	zap.L().Info("Successfully created container with ID " + imageDefinition.ContainerName + " and snapshot with ID " + imageDefinition.ContainerName + "-snapshot")
	zap.L().Info("Attempting to create a new task for container: " + imageDefinition.ContainerName)

	defer container.Delete(ctx, containerd.WithSnapshotCleanup)
	// Create and run a task after creating a container
	RunTask(container, imageDefinition.ContainerName)

	return container, err
}

func RunTask(container containerd.Container, containerName string) error {
	_, ctx, _ := ContainerdClient()
	// Create a new task with the container passed in as a parameter
	// Note, this is not an actual running container. We need to create a 'task' to run the container
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		zap.L().Error("An error occurred when trying to create a task from container: " + imageDefinition.ContainerName)
		return err
	}
	defer task.Delete(ctx)
	// Run this as a goroutine to avoid blocking
	// This runs it in the background - but we can still see the container/task stdout/stderr in our Go process
	go func() error {
		// See https://github.com/containerd/containerd/blob/main/docs/getting-started.md
		// We use `Wait` to avoid issues with processes that exit/complete very quick
		exitStatusC, err := task.Wait(ctx)
		if err != nil {
			zap.L().Error("An error occurred when trying to use `task.Wait` on task: " + task.ID())
			return err
		}

		if err := task.Start(ctx); err != nil {
			zap.L().Error("An error occurred when trying to start a task with: " + task.ID())
			return err
		}

		// Task succesfully created
		zap.L().Info("Successfully created task with ID " + task.ID())

		status := <-exitStatusC
		code, _, err := status.Result()
		if err != nil {
			return err
		}
		zap.L().Info("Task " + task.ID() + " exited with status code: " + string(code))

		return err
	}()

	return err
}

func ListImages() ([]Image, error) {
	var imageArray []Image
	client, ctx, _ := ContainerdClient()
	images, err := client.ListImages(ctx)

	if err != nil {
		zap.L().Error("An error occurred when using the containerd client..")
	}

	for _, image := range images {
		zap.L().Info(image.Name())
		imageArray = append(imageArray, Image{
			Name: image.Name(),
		})
	}

	return imageArray, err
}
