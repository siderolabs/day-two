// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package cmd provides d2ctl and all of the subcommands for it
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// RootCmd describes the root command for argesctl.
var RootCmd = &cobra.Command{
	Use:   "day-two",
	Short: "day-two deploys our amazing day-two offerings as helm charts.",
}

// Execute executes the root command.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
