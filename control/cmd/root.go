package cmd

import (
	config "csb/control/config"
	"fmt"
	"math/rand"
	"os"

	"github.com/spf13/cobra"
)

// placed under $HOME/.benchctl/config.json
const DEFAULT_CONFIG_DIR = ".benchctl"
const DEFAULT_CONFIG_FILE = "config.json"

type GlobalConfig struct {
	rg        *rand.Rand // default random generator for all the subcommands
	ctlConfig *config.BenchctlConfig
}

var GConfig *GlobalConfig = &GlobalConfig{}

var rootCmd = &cobra.Command{
	Use:   "benchctl [command] [flags]",
	Short: "Benchctl is a CLI tool for managing benchmarking tasks",
	Long:  "A CLI tool for managing benchmarking tasks including preparing, running, monitoring, and collecting results",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(0)
		}
	},
}

func init() {
	init_config()
	rootCmd.AddCommand(LoadCmd)
}

func init_config() {
	homedir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Failed to access user's HOME directory", err)
		os.Exit(1)
	}
	configDir := homedir + "/" + DEFAULT_CONFIG_DIR
	if _, err := os.Stat(configDir); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(configDir, 0755); err != nil {
				fmt.Println("Failed to create config directory: ", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Failed to check config directory: ", err)
			os.Exit(1)
		}
	}
	init_config_file(configDir)
}

func init_config_file(dirname string) {
	configFilePath := dirname + "/" + DEFAULT_CONFIG_FILE

	if _, err := os.Stat(configFilePath); err != nil {
		if os.IsNotExist(err) {
			defaultConfig := config.GetDefaultConfig()
			GConfig.rg = rand.New(rand.NewSource(defaultConfig.Seed))
			GConfig.ctlConfig = defaultConfig
			err = defaultConfig.WriteConfig(configFilePath)
			if err != nil {
				fmt.Println("Failed to write default config file: ", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Failed to check config file: ", err)
			os.Exit(1)
		}
	} else {
		localConfig, err := config.ReadConfig(configFilePath)
		if err != nil {
			fmt.Println("Failed to read config file: ", err)
			os.Exit(1)
		}
		GConfig.rg = rand.New(rand.NewSource(localConfig.Seed))
		GConfig.ctlConfig = localConfig
	}
}

func Execute() error {
	return rootCmd.Execute()
}
