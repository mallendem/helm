/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"io"
	"log"
	"log/slog"

	"github.com/spf13/cobra"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cmd/require"
)

const showDesc = `
This command consists of multiple subcommands to display information about a chart
`

const showAllDesc = `
This command inspects a chart (directory, file, or URL) and displays all its content
(values.yaml, Chart.yaml, README)
`

const showValuesDesc = `
This command inspects a chart (directory, file, or URL) and displays the contents
of the values.yaml file
`

const showChartDesc = `
This command inspects a chart (directory, file, or URL) and displays the contents
of the Chart.yaml file
`

const readmeChartDesc = `
This command inspects a chart (directory, file, or URL) and displays the contents
of the README file
`

const showCRDsDesc = `
This command inspects a chart (directory, file, or URL) and displays the contents
of the CustomResourceDefinition files
`

func newShowCmd(cfg *action.Configuration, out io.Writer) *cobra.Command {
	client := action.NewShow(action.ShowAll, cfg)

	showCommand := &cobra.Command{
		Use:     "show",
		Short:   "show information of a chart",
		Aliases: []string{"inspect"},
		Long:    showDesc,
		Args:    require.NoArgs,
	}

	// Function providing dynamic auto-completion
	validArgsFunc := func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return noMoreArgsComp()
		}
		return compListCharts(toComplete, true)
	}

	all := &cobra.Command{
		Use:               "all [CHART]",
		Short:             "show all information of the chart",
		Long:              showAllDesc,
		Args:              require.ExactArgs(1),
		ValidArgsFunction: validArgsFunc,
		RunE: func(_ *cobra.Command, args []string) error {
			client.OutputFormat = action.ShowAll
			err := addRegistryClient(client)
			if err != nil {
				return err
			}
			output, err := runShow(args, client)
			if err != nil {
				return err
			}
			fmt.Fprint(out, output)
			return nil
		},
	}

	valuesSubCmd := &cobra.Command{
		Use:               "values [CHART]",
		Short:             "show the chart's values",
		Long:              showValuesDesc,
		Args:              require.ExactArgs(1),
		ValidArgsFunction: validArgsFunc,
		RunE: func(_ *cobra.Command, args []string) error {
			client.OutputFormat = action.ShowValues
			err := addRegistryClient(client)
			if err != nil {
				return err
			}
			output, err := runShow(args, client)
			if err != nil {
				return err
			}
			fmt.Fprint(out, output)
			return nil
		},
	}

	chartSubCmd := &cobra.Command{
		Use:               "chart [CHART]",
		Short:             "show the chart's definition",
		Long:              showChartDesc,
		Args:              require.ExactArgs(1),
		ValidArgsFunction: validArgsFunc,
		RunE: func(_ *cobra.Command, args []string) error {
			client.OutputFormat = action.ShowChart
			err := addRegistryClient(client)
			if err != nil {
				return err
			}
			output, err := runShow(args, client)
			if err != nil {
				return err
			}
			fmt.Fprint(out, output)
			return nil
		},
	}

	readmeSubCmd := &cobra.Command{
		Use:               "readme [CHART]",
		Short:             "show the chart's README",
		Long:              readmeChartDesc,
		Args:              require.ExactArgs(1),
		ValidArgsFunction: validArgsFunc,
		RunE: func(_ *cobra.Command, args []string) error {
			client.OutputFormat = action.ShowReadme
			err := addRegistryClient(client)
			if err != nil {
				return err
			}
			output, err := runShow(args, client)
			if err != nil {
				return err
			}
			fmt.Fprint(out, output)
			return nil
		},
	}

	crdsSubCmd := &cobra.Command{
		Use:               "crds [CHART]",
		Short:             "show the chart's CRDs",
		Long:              showCRDsDesc,
		Args:              require.ExactArgs(1),
		ValidArgsFunction: validArgsFunc,
		RunE: func(_ *cobra.Command, args []string) error {
			client.OutputFormat = action.ShowCRDs
			err := addRegistryClient(client)
			if err != nil {
				return err
			}
			output, err := runShow(args, client)
			if err != nil {
				return err
			}
			fmt.Fprint(out, output)
			return nil
		},
	}

	cmds := []*cobra.Command{all, readmeSubCmd, valuesSubCmd, chartSubCmd, crdsSubCmd}
	for _, subCmd := range cmds {
		addShowFlags(subCmd, client)
		showCommand.AddCommand(subCmd)
	}

	return showCommand
}

func addShowFlags(subCmd *cobra.Command, client *action.Show) {
	f := subCmd.Flags()

	f.BoolVar(&client.Devel, "devel", false, "use development versions, too. Equivalent to version '>0.0.0-0'. If --version is set, this is ignored")
	if subCmd.Name() == "values" {
		f.StringVar(&client.JSONPathTemplate, "jsonpath", "", "supply a JSONPath expression to filter the output")
	}
	addChartPathOptionsFlags(f, &client.ChartPathOptions)

	err := subCmd.RegisterFlagCompletionFunc("version", func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 1 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return compVersionFlag(args[0], toComplete)
	})

	if err != nil {
		log.Fatal(err)
	}
}

func runShow(args []string, client *action.Show) (string, error) {
	slog.Debug("original chart version", "version", client.Version)
	if client.Version == "" && client.Devel {
		slog.Debug("setting version to >0.0.0-0")
		client.Version = ">0.0.0-0"
	}

	cp, err := client.LocateChart(args[0], settings)
	if err != nil {
		return "", err
	}
	return client.Run(cp)
}

func addRegistryClient(client *action.Show) error {
	registryClient, err := newRegistryClient(client.CertFile, client.KeyFile, client.CaFile,
		client.InsecureSkipTLSverify, client.PlainHTTP, client.Username, client.Password)
	if err != nil {
		return fmt.Errorf("missing registry client: %w", err)
	}
	client.SetRegistryClient(registryClient)
	return nil
}
