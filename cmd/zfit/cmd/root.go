package cmd

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/integralist/go-findroot/find"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var cfgFile string
var weightKG float64

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zfit",
	Short: "",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.zfit.yaml)")
	rootCmd.PersistentFlags().Float64Var(&weightKG, "kg", 0.0, "weight in kg")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
func GitRootDir() string {
	root, err := find.Repo()
	if err != nil {
		return ""
	}
	return root.Path
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		usr, _ := user.Current()
		home := usr.HomeDir

		// Search config in home directory with name ".dbsnapper" (without extension).
		viper.AddConfigPath(GitRootDir())
		viper.AddConfigPath(home)

		viper.SetConfigName(".zfit")
	}

	viper.SetEnvPrefix("zfit")
	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.WithFields(
			log.Fields{
				"file": viper.ConfigFileUsed(),
			},
		).Debugln("Using Config File:")
	}

	if viper.GetBool("debug") == true {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}
