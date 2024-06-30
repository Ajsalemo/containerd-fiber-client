package ctr

import (
	"context"
	"syscall"
	"time"

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
	CtrTaskDef      containerd.Task
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

	zap.L().Info("Attempting to create a new container for: " + imageDefinition.ContainerName)
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
	}
	// Create a new task with the container passed in as a parameter
	// Note, this is not an actual running container. We need to create a 'task' to run the container
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	ctrImageProps.CtrTaskDef = task

	if err != nil {
		zap.L().Error("An error occurred when trying to create a task from container: " + imageDefinition.ContainerName)
		return err
	}
	// Run this as a goroutine to avoid blocking
	// This runs it in the background - but we can still see the container/task stdout/stderr in our Go process
	go func() error {
		// See https://github.com/containerd/containerd/blob/main/docs/getting-started.md
		// We use `Wait` to avoid issues with processes that exit/complete very quick
		exitStatusC, err := ctrImageProps.CtrTaskDef.Wait(ctx)
		if err != nil {
			zap.L().Error("An error occurred when trying to use `ctrImageProps.CtrTaskDef.Wait` on task: " + ctrImageProps.CtrTaskDef.ID())
			return err
		}

		if err := ctrImageProps.CtrTaskDef.Start(ctx); err != nil {
			zap.L().Error("An error occurred when trying to start a task with: " + ctrImageProps.CtrTaskDef.ID())
			return err
		}

		// Task succesfully created
		zap.L().Info("Successfully created task with ID " + ctrImageProps.CtrTaskDef.ID())

		status := <-exitStatusC
		code, _, err := status.Result()
		if err != nil {
			return err
		}
		zap.L().Info("Task " + ctrImageProps.CtrTaskDef.ID() + " exited with status code: " + string(code))

		return err
	}()

	return err
}

// StopTask deletes the task
func StopTask() error {
	_, ctx, err := ContainerdClient()

	if err != nil {
		zap.L().Error("An error occurred when trying to use the containerd client..")
		zap.L().Error(err.Error())
		return err
	}

	if err != nil {
		zap.L().Error("An error occurred when trying to use the containerd client..")
		zap.L().Error(err.Error())
		return err
	}

	zap.L().Info("Attempting to kill task with ID " + ctrImageProps.CtrTaskDef.ID() + " for container: " + ctrImageProps.CtrContainerDef.ID() + " with SIGKTERM")
	// Kill the task
	if err := ctrImageProps.CtrTaskDef.Kill(ctx, syscall.SIGTERM); err != nil {
		zap.L().Error("An error occurred when trying to kill task: " + ctrImageProps.CtrTaskDef.ID() + " with SIGTERM")
		zap.L().Error(err.Error())
		return err
	}

	taskStatus, err := ctrImageProps.CtrTaskDef.Status(ctx)
	if err != nil {
		zap.L().Error("An error occurred when trying to get the status of task: " + ctrImageProps.CtrTaskDef.ID())
		zap.L().Error(err.Error())
		return err
	}
	// Check if the task is still running - if so, wait 30 seconds and kill it with SIGKILL
	// Poll every 5 seconds to check if the task is still running
	go func() error {
		if taskStatus.Status == "running" {
			zap.L().Info("Checking if Task " + ctrImageProps.CtrTaskDef.ID() + " is still running or respected SIGTERM..")
			zap.L().Info("Task " + ctrImageProps.CtrTaskDef.ID() + " is still running..")
			for range time.Tick(5 * time.Second) {
				taskStatus, err := ctrImageProps.CtrTaskDef.Status(ctx)
				if err != nil {
					zap.L().Error("An error occurred when trying to get the status of task: " + ctrImageProps.CtrTaskDef.ID())
					zap.L().Error(err.Error())
					return err
				}

				zap.L().Info("Task status for " + ctrImageProps.CtrTaskDef.ID() + ": " + string(taskStatus.Status))
				// If the task is no longer running, break out of the loop
				if taskStatus.Status != "running" {
					zap.L().Info("Task status for " + ctrImageProps.CtrTaskDef.ID() + ": " + string(taskStatus.Status))
					break
				}
			}

			// Kill the task
			if err := ctrImageProps.CtrTaskDef.Kill(ctx, syscall.SIGKILL); err != nil {
				zap.L().Error("An error occurred when trying to kill task: " + ctrImageProps.CtrTaskDef.ID() + " with SIGKILL")
				zap.L().Error(err.Error())
				return err
			}
			// Delete the task after killing it
			exitStatusK, err := ctrImageProps.CtrTaskDef.Delete(ctx)

			if err != nil {
				zap.L().Error("An error occurred when trying to delete task: " + ctrImageProps.CtrTaskDef.ID())
				zap.L().Error(err.Error())
				return err
			}
			// Check the exit status of the task
			code, _, err := exitStatusK.Result()
			if err != nil {
				zap.L().Error("An error occurred when trying to read the exit status of task: " + ctrImageProps.CtrTaskDef.ID())
				zap.L().Error(err.Error())
				return err
			}
			zap.L().Info("Task " + ctrImageProps.CtrTaskDef.ID() + " exited with status code: " + string(code))

			zap.L().Info("Deleting container and snapshot for task: " + ctrImageProps.CtrTaskDef.ID() + " for container: " + ctrImageProps.CtrContainerDef.ID())
			// Delete the container and snapshot after stopping the task
			cerr := ctrImageProps.CtrContainerDef.Delete(ctx, containerd.WithSnapshotCleanup)

			if cerr != nil {
				zap.L().Error("An error occurred when trying to delete container: " + ctrImageProps.CtrContainerDef.ID())
				zap.L().Error(cerr.Error())
				return err
			}
		}

		return err
	}()

	zap.L().Info("Succesfully stopped task with ID " + ctrImageProps.CtrTaskDef.ID() + " for container: " + ctrImageProps.CtrContainerDef.ID())

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
