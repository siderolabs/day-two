package pulumi

import (
	"context"
	"fmt"
	"os"
	"time"

	helm "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
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

	fmt.Printf("created/selected stack %q\n", stackName)

	w := s.Workspace()

	fmt.Println("installing the k8s plugin")

	// for inline source programs, we must manage plugins ourselves
	err = w.InstallPlugin(ctx, "kubernetes", "v3.15.1")
	if err != nil {
		return err
	}

	fmt.Println("successfully installed k8s plugin")

	fmt.Println("successfully set config")
	fmt.Println("starting refresh")

	_, err = s.Refresh(ctx)
	if err != nil {
		return err
	}

	fmt.Println("refresh succeeded!")

	fmt.Println("starting update")

	// wire up our update to stream progress to stdout
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	_, err = s.Up(ctx, stdoutStreamer)
	if err != nil {
		return err
	}

	fmt.Println("update succeeded!")

	return nil
}

func deployCharts(configPath string) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		charts, err := config.LoadConfig(configPath)
		if err != nil {
			return err
		}

		deployedCharts := map[string]*helm.Release{}

		dependencyAttempts := 0

		for len(charts.Charts) > 0 {
			for name, chart := range charts.Charts {

				depList := []pulumi.Resource{}

				missingDeps := false

				for _, dep := range chart.Dependencies {
					relPtr, ok := deployedCharts[dep]
					if !ok {
						fmt.Printf("chart %s missing dependency %s, skipping for now\n", name, dep)

						missingDeps = true
						break
					}

					depList = append(depList, relPtr)
				}

				if missingDeps {
					dependencyAttempts++
					continue
				}

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
						pulumi.NewFileAsset(chart.ValuesPath),
					}
				}

				helmPtr, err := helm.NewRelease(ctx, name, releaseArgs, pulumi.DependsOn(depList))
				if err != nil {
					return err
				}

				deployedCharts[name] = helmPtr

				delete(charts.Charts, name)
			}

			if dependencyAttempts > 10 {
				return fmt.Errorf("failed to resolve dependencies for chart")
			}

			time.Sleep(30 * time.Second)
		}

		return nil
	}
}
