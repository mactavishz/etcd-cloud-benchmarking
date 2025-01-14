package cmd

import (
	benchCfg "csb/control/config"
	config "csb/control/config"
	constants "csb/control/constants"
	"fmt"
	"math/rand"
	"os"
	"path"

	"github.com/spf13/cobra"
)

type GlobalConfig struct {
	rg            *rand.Rand // default random generator for all the subcommands
	ctlConfig     *config.BenchctlConfig
	ctlConfigPath string
}

func (g *GlobalConfig) UpdateRg(newSeed int64) {
	g.rg = rand.New(rand.NewSource(newSeed))
}

func (g *GlobalConfig) GetKeyFilePath() string {
	return path.Join(g.ctlConfigPath, constants.DEFAULT_KEY_FILE)
}

func (g *GlobalConfig) GetConfigFilePath() string {
	return path.Join(g.ctlConfigPath, constants.DEFAULT_CONFIG_FILE)
}

var GConfig *GlobalConfig = &GlobalConfig{}

var rootCmd = &cobra.Command{
	Use:   "benchctl [command] [flags]",
	Short: "Benchctl is a CLI tool for managing benchmarking tasks",
	Long:  "A CLI tool for managing benchmarking tasks including preparing, running, monitoring, and collecting results",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			err := cmd.Help()
			if err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
	},
}

func init() {
	initConfigPath()
	loadConfig()
	rootCmd.AddCommand(RunCmd)
	rootCmd.AddCommand(ConfigCmd)
}

func initConfigPath() {
	homedir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Failed to access user's HOME directory", err)
		os.Exit(1)
	}
	configDir := path.Join(homedir, constants.DEFAULT_CONFIG_DIR)
	GConfig.ctlConfigPath = configDir
}

func loadConfig() {
	configFilePath := path.Join(GConfig.ctlConfigPath, constants.DEFAULT_CONFIG_FILE)
	if _, err := os.Stat(configFilePath); err != nil {
		if os.IsNotExist(err) {
			return
		} else {
			fmt.Println("Failed to check config directory: ", err)
		}
	}
	localConfig, err := benchCfg.ReadConfig(configFilePath)
	if err != nil {
		fmt.Println("Failed to read config file: ", err)
		return
	}

	GConfig.UpdateRg(localConfig.Seed)
	GConfig.ctlConfig = localConfig
}

func Execute() error {
	return rootCmd.Execute()
}
