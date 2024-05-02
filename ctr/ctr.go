package ctr

import (
	"context"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
)

func ContainerdClient() (*containerd.Client, context.Context, error) {
	client, err := containerd.New("/run/containerd/containerd.sock")
	// Set a default namespace
	ctx := namespaces.WithNamespace(context.Background(), "default")

	return client, ctx, err
}