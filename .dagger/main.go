// A generated module for Particulate functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.
package main

import (
	"context"

	"dagger/particulate/internal/dagger"
)

type Particulate struct{}

func (m *Particulate) Lint(
	ctx context.Context,
	// +defaultPath="/"
	source *dagger.Directory,
) (string, error) {
	return dag.Container().
		From("golangci/golangci-lint:v2.1.6").
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"golangci-lint", "run", "./..."}).
		Stdout(ctx)
}

// Build and publish multi-platform image
func (m *Particulate) Build(
	ctx context.Context,
	// +defaultPath="/"
	src *dagger.Directory,
) (string, error) {
	// platforms to build for and push in a multi-platform image
	var platforms = []dagger.Platform{
		"linux/arm64", // a.k.a. aarch64
	}

	platformVariants := make([]*dagger.Container, 0, len(platforms))
	for _, platform := range platforms {

		// pull golang image for this platform
		ctr := dag.Container(dagger.ContainerOpts{Platform: platform}).
			From("golang:alpine").
			WithDirectory("/src", src).
			WithDirectory("/output", dag.Directory()).
			WithEnvVariable("CGO_ENABLED", "0").
			WithWorkdir("/src").
			WithExec([]string{"go", "build", "-o", "/output/airdata-articulate"})

		// select output directory
		outputDir := ctr.Directory("/output")

		// wrap the output directory in the new empty container marked
		// with the same platform
		binaryCtr := dag.Container(dagger.ContainerOpts{Platform: platform}).
			WithRootfs(outputDir).
			WithExposedPort(8000).
			WithEntrypoint([]string{"/airdata-articulate"})

		platformVariants = append(platformVariants, binaryCtr)
	}

	imageRepo := "ghcr.io/charemma/airdata-particulate:latest"

	// publish to registry
	imageDigest, err := dag.Container().
		Publish(ctx, imageRepo, dagger.ContainerPublishOpts{
			PlatformVariants: platformVariants,
		})

	if err != nil {
		return "", err
	}

	// return build directory
	return imageDigest, nil
}

// Deploy the application to my Kubernetes cluster
func (m *Particulate) Deploy(
	ctx context.Context,
	kubeconfig *dagger.File,
	// +defaultPath="/"
	source *dagger.Directory,
) (string, error) {

	_, err := m.Build(ctx, source)

	if err != nil {
		return "", err
	}

	return dag.Container().
		From("bitnami/kubectl:latest").
		WithEnvVariable("HOME", "/kube").
		WithDirectory("/manifest", source).
		WithMountedFile("/kube/config", kubeconfig).
		WithEnvVariable("KUBECONFIG", "/kube/config").
		WithExec([]string{"kubectl", "apply", "-f", "/manifest/k8s/deployment.yml", "-n", "monitoring"}).
		Stdout(ctx)
}
