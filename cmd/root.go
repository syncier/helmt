/*
Copyright © 2020 Syncier GmbH <info@syncier.com>

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
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/syncier/helmt/pkg/helmt"
)

var cfgFile string
var clean bool

var rootCmd = &cobra.Command{
	Use:   "helmt <filename>",
	Short: "A simple wrapper around helm template",
	Long: `A simple wrapper around helm template
It expects a filename which contains all necessary information:

chart: jenkins
version: 2.0.0
repository: https://kubernetes-charts.storage.googleapis.com
namespace: jenkins
values:
  - values1.yaml
  - values2.yaml

namespace and values are optional
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := "helm-chart.yaml"

		if len(args) == 1 {
			filename = args[0]
		}
		fmt.Printf("templating '%s'\n", filename)

		return helmt.HelmTemplate(filename, clean)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.helmt.yaml)")
	rootCmd.PersistentFlags().BoolVar(&clean, "clean", false, "delete existing templates before rendering")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".helmt" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".helmt")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
