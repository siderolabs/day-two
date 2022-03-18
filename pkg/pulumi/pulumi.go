// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package pulumi provides the pulumi interface for d2ctl
package pulumi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	helm "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/talos-systems/day-two/pkg/config"
)

// Up creates or updates the chart resources from the config file.
func Up(ctx context.Context, configPath, statePath string) error {
	projectName := "day-two"
	stackName := "day-two"

	err := ensureStateDirExists(statePath)
	if err != nil {
		return err
	}

	// Specify a local backend instead of using the service.
	project := auto.Project(workspace.Project{
		Name:    tokens.PackageName(projectName),
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
		Backend: &workspace.ProjectBackend{
			URL: "file://" + statePath,
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

//nolint:gocognit
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
					Name:            pulumi.StringPtr(name),
					Namespace:       pulumi.String(chart.Namespace),
					RepositoryOpts: &helm.RepositoryOptsArgs{
						Repo: pulumi.String(chart.Repo),
					},
				}

				if chart.ValuesPath != "" {
					valuesPath, err := filepath.Abs(chart.ValuesPath)
					if err != nil {
						return err
					}

					releaseArgs.ValueYamlFiles = pulumi.AssetOrArchiveArray{
						pulumi.NewFileAsset(valuesPath),
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

func ensureStateDirExists(statePath string) error {
	// Ensure path exists (create it if not)
	absStatePath, err := filepath.Abs(statePath)
	if err != nil {
		return err
	}

	info, err := os.Stat(absStatePath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(absStatePath, os.FileMode(0o700))
		if err != nil {
			return err
		}

		info, err = os.Stat(absStatePath)
	}

	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("state-path '%s' is not a directory", absStatePath)
	}

	return nil
}
