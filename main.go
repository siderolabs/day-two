package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	// define our program that creates our pulumi resources.
	// we refer to this style as "inline" pulumi programs where both program + automation can be compiled in the same
	// binary. no need for separate projects.
	deployFunc := func(ctx *pulumi.Context) error {
		// Get PWD so we have full path to values files. Seems to fail with relative :shrugs:
		path, err := os.Getwd()
		if err != nil {
			return err
		}

		_, err = helm.NewRelease(ctx, "loki", &helm.ReleaseArgs{
			Chart:           pulumi.String("loki-stack"),
			CreateNamespace: pulumi.BoolPtr(true),
			Namespace:       pulumi.String("loki"),
			RepositoryOpts: &helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://grafana.github.io/helm-charts"),
			},
			ValueYamlFiles: pulumi.AssetOrArchiveArray{
				pulumi.NewFileAsset(filepath.Join(path, "loki/values.yaml")),
			},
		})
		if err != nil {
			return err
		}

		metallb, err := helm.NewRelease(ctx, "metallb", &helm.ReleaseArgs{
			Chart:           pulumi.String("metallb"),
			CreateNamespace: pulumi.BoolPtr(true),
			Namespace:       pulumi.String("metallb"),
			RepositoryOpts: &helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://metallb.github.io/metallb"),
			},
			ValueYamlFiles: pulumi.AssetOrArchiveArray{
				pulumi.NewFileAsset(filepath.Join(path, "metallb/values.yaml")),
			},
		})
		if err != nil {
			return err
		}

		_, err = helm.NewRelease(
			ctx,
			"ingress-nginx",
			&helm.ReleaseArgs{
				Chart:           pulumi.String("ingress-nginx"),
				CreateNamespace: pulumi.BoolPtr(true),
				Namespace:       pulumi.String("ingress"),
				RepositoryOpts: &helm.RepositoryOptsArgs{
					Repo: pulumi.String("https://kubernetes.github.io/ingress-nginx"),
				},
			},
			pulumi.DependsOn([]pulumi.Resource{metallb}),
		)
		if err != nil {
			return err
		}

		_, err = helm.NewRelease(
			ctx,
			"cert-manager",
			&helm.ReleaseArgs{
				Chart:           pulumi.String("cert-manager"),
				CreateNamespace: pulumi.BoolPtr(true),
				Namespace:       pulumi.String("cert-manager"),
				RepositoryOpts: &helm.RepositoryOptsArgs{
					Repo: pulumi.String("https://charts.jetstack.io"),
				},
				ValueYamlFiles: pulumi.AssetOrArchiveArray{
					pulumi.NewFileAsset(filepath.Join(path, "cert-manager/values.yaml")),
				},
			},
		)
		if err != nil {
			return err
		}

		return nil
	}

	ctx := context.Background()

	projectName := "day-two"
	stackName := "day-two"

	// Specify a local backend instead of using the service.
	project := auto.Project(workspace.Project{
		Name:    tokens.PackageName(projectName),
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
		Backend: &workspace.ProjectBackend{
			URL: "file://~/.pulumi-local",
		},
	})

	// Setup a passphrase secrets provider and use an environment variable to pass in the passphrase.
	secretsProvider := auto.SecretsProvider("passphrase")
	envvars := auto.EnvVars(map[string]string{
		// In a real program, you would feed in the password securely or via the actual environment.
		"PULUMI_CONFIG_PASSPHRASE": "password",
	})

	stackSettings := auto.Stacks(map[string]workspace.ProjectStack{
		stackName: {SecretsProvider: "passphrase"},
	})

	// create or select a stack matching the specified name and project.
	// this will set up a workspace with everything necessary to run our inline program (deployFunc)
	s, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, deployFunc, project, secretsProvider, stackSettings, envvars)
	if err != nil {
		fmt.Printf("Failed to upsert stack: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created/Selected stack %q\n", stackName)

	w := s.Workspace()

	fmt.Println("Installing the k8s plugin")

	// for inline source programs, we must manage plugins ourselves
	err = w.InstallPlugin(ctx, "kubernetes", "v3.15.1")
	if err != nil {
		fmt.Printf("Failed to install program plugins: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully installed k8s plugin")

	fmt.Println("Successfully set config")
	fmt.Println("Starting refresh")

	_, err = s.Refresh(ctx)
	if err != nil {
		fmt.Printf("Failed to refresh stack: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Refresh succeeded!")

	fmt.Println("Starting update")

	// wire up our update to stream progress to stdout
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	_, err = s.Up(ctx, stdoutStreamer)
	if err != nil {
		fmt.Printf("Failed to update stack: %v\n\n", err)
		os.Exit(1)
	}

	fmt.Println("Update succeeded!")

}
