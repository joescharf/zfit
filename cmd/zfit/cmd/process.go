package cmd

import (
	"os"

	"github.com/joescharf/zfit/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// processCmd represents the process command
var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Process a .fit file",
	Args:  cobra.MinimumNArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		process(args)
	},
}

var csvFile *os.File

func init() {
	rootCmd.AddCommand(processCmd)
}

var DATA zfit.Data
var RESULTS zfit.Results

func process(args []string) {
	z := zfit.New(args[0])

	// Check the weightKG parameter. if Zero, pull the data from config file.
	if weightKG == 0.0 {
		weightKG = viper.GetFloat64("kg")
	}
	z.SetWeight(weightKG)
	z.CritPower()
	z.PrintBasicStats()
	z.PrintCritPowerResultsTable()
	z.CritPowerToCSV()
}
