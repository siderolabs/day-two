package pulumi

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

	"github.com/talos-systems/day-two/pkg/config"
)

func Up(ctx context.Context, configPath string) error {
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
		"PULUMI_CONFIG_PASSPHRASE": "password",
	})

	stackSettings := auto.Stacks(map[string]workspace.ProjectStack{
		stackName: {SecretsProvider: "passphrase"},
	})

	// create or select a stack matching the specified name and project.
	// this will set up a workspace with everything necessary to run our inline program (deployFunc)
	s, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, deployCharts(configPath), project, secretsProvider, stackSettings, envvars)
	if err != nil {
		return err
	}

	fmt.Printf("Created/Selected stack %q\n", stackName)

	w := s.Workspace()

	fmt.Println("Installing the k8s plugin")

	// for inline source programs, we must manage plugins ourselves
	err = w.InstallPlugin(ctx, "kubernetes", "v3.15.1")
	if err != nil {
		return err
	}

	fmt.Println("Successfully installed k8s plugin")

	fmt.Println("Successfully set config")
	fmt.Println("Starting refresh")

	_, err = s.Refresh(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Refresh succeeded!")

	fmt.Println("Starting update")

	// wire up our update to stream progress to stdout
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	_, err = s.Up(ctx, stdoutStreamer)
	if err != nil {
		return err
	}

	fmt.Println("Update succeeded!")

	return nil
}

func deployCharts(configPath string) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		// Get PWD so we have full path to values files. Seems to fail with relative :shrugs:
		path, err := os.Getwd()
		if err != nil {
			return err
		}

		chartList, err := config.LoadConfig(configPath)
		if err != nil {
			return err
		}

		for _, chart := range chartList.Charts {
			releaseArgs := &helm.ReleaseArgs{
				Chart:           pulumi.String(chart.Chart),
				CreateNamespace: pulumi.BoolPtr(true),
				Namespace:       pulumi.String(chart.Namespace),
				RepositoryOpts: &helm.RepositoryOptsArgs{
					Repo: pulumi.String(chart.Repo),
				},
			}

			if chart.ValuesPath != "" {
				releaseArgs.ValueYamlFiles = pulumi.AssetOrArchiveArray{
					pulumi.NewFileAsset(
						filepath.Join(path, chart.ValuesPath),
					),
				}
			}

			// TODO support dependencies if needed
			_, err = helm.NewRelease(ctx, chart.Name, releaseArgs)
			if err != nil {
				return err
			}
		}

		return nil
	}
}
