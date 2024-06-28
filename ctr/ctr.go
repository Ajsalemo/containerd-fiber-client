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
}

type CtrImageProps struct {
	CtrImageDef     containerd.Image
	CtrContainerDef containerd.Container `json:"ctrContainerDef"`
	PulledImage     string               `json:"pulledImage"`
}

type Image struct {
	Name string
}

var ctrImageProps CtrImageProps

func ContainerdClient() (*containerd.Client, context.Context, error) {
	client, err := containerd.New("/run/containerd/containerd.sock")
	ctx := namespaces.WithNamespace(context.Background(), "default")

	return client, ctx, err
}

// Function to pull authenticated images
// The primary difference here is the use of `Resolver` to handle authentication
func PullAuthenticatedImage(imageDefinition ImageDefintion, resolver remotes.Resolver) error {
	client, ctx, err := ContainerdClient()

	if err != nil {
		zap.L().Error("An error occurred when trying to use the containerd client..")
		zap.L().Error(err.Error())
		return err
	}

	image, err := client.Pull(ctx, imageDefinition.Registry+"/"+imageDefinition.Image+":"+imageDefinition.Tag, containerd.WithPullUnpack, containerd.WithResolver(resolver))

	if err != nil {
		zap.L().Error("An error occurred when trying to pull an authenticated image..")
		zap.L().Error(err.Error())
		return err
	}
	// We set our struct properties to these containerd `image` (containerd.Image) values to be used around in other functions
	ctrImageProps.CtrImageDef = image
	ctrImageProps.PulledImage = image.Name()

	zap.L().Info("Succesfully pulled image " + image.Name())

	return err
}

// Function to pull public images
// No authentication
func PullPublicImage(imageDefinition ImageDefintion) error {
	client, ctx, err := ContainerdClient()
	if err != nil {
		zap.L().Error("An error occurred when trying to use the containerd client..")
		zap.L().Error(err.Error())
		return err
	}	
	
	image, err := client.Pull(ctx, imageDefinition.Registry+"/"+imageDefinition.Image+":"+imageDefinition.Tag, containerd.WithPullUnpack)

	if err != nil {
		zap.L().Error("An error occurred when trying to pull a public image..")
		return err
	}
	// We set our struct properties to these containerd `image` (containerd.Image) values to be used around in other functions
	ctrImageProps.CtrImageDef = image
	ctrImageProps.PulledImage = image.Name()

	zap.L().Info("Succesfully pulled image " + image.Name())

	return err
}

func CreateContainer(imageDefinition ImageDefintion) error {
	client, ctx, err := ContainerdClient()

	if err != nil {
		zap.L().Error("An error occurred when trying to use the containerd client..")
		zap.L().Error(err.Error())
		return err
	}
	// Create a new container with the image
	// Note, this is not an actual running container. We need to create a 'task' to run the container
	container, err := client.NewContainer(
		ctx,
		imageDefinition.ContainerName,
		containerd.WithNewSnapshot(imageDefinition.ContainerName+"-snapshot", ctrImageProps.CtrImageDef),
		containerd.WithNewSpec(oci.WithImageConfig(ctrImageProps.CtrImageDef)),
	)

	if err != nil {
		zap.L().Error("An error occurred when trying to create a new container for image: " + imageDefinition.Image)
		zap.L().Error(err.Error())
		return err
	}
	// We set our struct properties to these containerd `container` (containerd.Container) values to be used around in other functions
	ctrImageProps.CtrContainerDef = container

	zap.L().Info("Successfully created container with ID " + imageDefinition.ContainerName + " and snapshot with ID " + imageDefinition.ContainerName + "-snapshot")
	zap.L().Info("Attempting to create a new task for container: " + imageDefinition.ContainerName)

	defer container.Delete(ctx, containerd.WithSnapshotCleanup)
	// Create and run a task after creating a container
	RunTask(container, imageDefinition)

	return err
}

func RunTask(container containerd.Container, imageDefinition ImageDefintion) error {
	_, ctx, err := ContainerdClient()

	if err != nil {
		zap.L().Error("An error occurred when trying to use the containerd client..")
		zap.L().Error(err.Error())
		return err
	}	// Create a new task with the container passed in as a parameter
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

// Function to list currently pulled images
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
