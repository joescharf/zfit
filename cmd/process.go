package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/mxmCherry/movavg"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tormoder/fit"
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

type Average struct {
	Timestamp time.Time `csv:"timestamp"`
	T         int       `csv:"t"`
	Pwr       uint16    `csv:"pwr"`
	HR        uint8     `csv:"hr"`
	Speed     uint16    `csv:"speed"`
	Cadence   uint8     `csv:"cadence"`
	FifteenS  float64   `csv:"15s"`
	ThirtyS   float64   `csv:"30s"`
	OneM      float64   `csv:"1m"`
	FiveM     float64   `csv:"5m"`
	TenM      float64   `csv:"10m"`
	TwentyM   float64   `csv:"20m"`
}

func process(args []string) {
	// process argument (filename)
	f := args[0]
	base := filepath.Base(f)
	ext := filepath.Ext(base)
	fileName := strings.Replace(base, ext, "", -1)
	fitFileName := filepath.Join("./", base)
	csvFileName := fmt.Sprintf("%s.csv", fileName)

	log.WithFields(log.Fields{
		"fileName":    fileName,
		"Ext":         ext,
		"FitFilename": fitFileName,
		"csvFilename": csvFileName,
	}).Debugln("Processing File: ")

	// Read our fit test file data
	fitFile, err := ioutil.ReadFile(fitFileName)
	if err != nil {
		log.Fatalln(err)
	}
	csvFile, err = os.OpenFile(csvFileName, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	defer csvFile.Close()

	// Decode the fitData file data
	fitData, err := fit.Decode(bytes.NewReader(fitFile))
	if err != nil {
		fmt.Println(err)
		return
	}

	// Get the actual activity
	activity, err := fitData.Activity()
	if err != nil {
		fmt.Println(err)
		return
	}

	/////////////

	fmt.Printf("FIT File: Type %s, Manufacturer: %s, Created: %v\n", fitData.Type(), fitData.FileId.Manufacturer, fitData.FileId.TimeCreated)
	fmt.Println("Sports:")
	// Print the sport of the Sessions
	for _, session := range activity.Sessions {
		fmt.Printf("  %s\n", session.Sport)
	}
	fmt.Printf("Activity data: Sessions: %v, Records: %v, Events: %v, Hrvs: %v\n", len(activity.Sessions), len(activity.Records), len(activity.Events), len(activity.Hrvs))
	fmt.Printf("Activity timestamp resolution: Timestamp 0: %v, Timestamp 1: %v, diff: %v\n", activity.Records[1000].Timestamp, activity.Records[1001].Timestamp, activity.Records[1].Timestamp.Unix()-activity.Records[0].Timestamp.Unix())

	// Session stuff:
	fmt.Println("")
	for i, sess := range activity.Sessions {
		fmt.Printf("Session Information for session %d:\n", i)
		fmt.Printf("  Power: Avg, Max, Normalized [%d, %d, %d]\n", sess.AvgPower, sess.MaxPower, sess.NormalizedPower)
		fmt.Printf("  HR: Avg, Max [%d, %d]\n", sess.AvgHeartRate, sess.MaxHeartRate)
		fmt.Printf("  TSS: %d\n\n", sess.TrainingStressScore)
	}

	for i, e := range activity.Events {
		fmt.Printf("Event Information for event %d:\n", i)
		fmt.Printf("  %v - EventType: %s\n\n", e.Timestamp, e.EventType.String())
	}

	// Multi moving average caluclation and csv output:
	mma(activity)

}

func mma(activity *fit.ActivityFile) {
	// Multi Moving average: 15s, 30s, 1m, 5m, 10m, 20m
	records := activity.Records[0:]
	powerArray := make([]float64, len(records))
	avgArray := make([][]float64, len(records))
	avgs := make([]*Average, len(records))
	maxes := make(map[string]float64)

	multi := movavg.Multi{
		movavg.NewSMA(15),
		movavg.NewSMA(30),
		movavg.NewSMA(60),
		movavg.NewSMA(300),
		movavg.NewSMA(600),
		movavg.NewSMA(1200),
	}
	// multiMA := movavg.MultiThreadSafe(multi)
	multiMA := movavg.MultiThreadSafe(multi)

	// Load the power data:
	for i, v := range records {
		powerArray[i] = float64(v.Power)
	}

	// Iterate on power data and get the moving averages:
	fmt.Println("")
	for i, v := range records {
		var k string
		var val float64

		avg := multiMA.Add(float64(v.Power))

		avgArray[i] = avg

		// Map the avg array:
		avgs[i] = &Average{
			T:         i,
			Timestamp: v.Timestamp,
			Pwr:       v.Power,
			HR:        v.HeartRate,
			Speed:     v.Speed,
			Cadence:   v.Cadence,
		}

		if i >= 1200 {
			k = "1200"
			val = avg[5]

			avgs[i].TwentyM = avg[5]
			avgs[i].TenM = avg[4]
			avgs[i].FiveM = avg[3]
			avgs[i].OneM = avg[2]
			avgs[i].ThirtyS = avg[1]
			avgs[i].FifteenS = avg[0]

		} else if i >= 600 {
			k = "600"
			val = avg[4]
			avgs[i].TenM = avg[4]
			avgs[i].FiveM = avg[3]
			avgs[i].OneM = avg[2]
			avgs[i].ThirtyS = avg[1]
			avgs[i].FifteenS = avg[0]
		} else if i >= 300 {
			k = "300"
			val = avg[3]
			avgs[i].FiveM = avg[3]
			avgs[i].OneM = avg[2]
			avgs[i].ThirtyS = avg[1]
			avgs[i].FifteenS = avg[0]
		} else if i >= 60 {
			k = "60"
			val = avg[2]
			avgs[i].OneM = avg[2]
			avgs[i].ThirtyS = avg[1]
			avgs[i].FifteenS = avg[0]
		} else if i >= 30 {
			k = "30"
			val = avg[1]
			avgs[i].ThirtyS = avg[1]
			avgs[i].FifteenS = avg[0]
		} else if i >= 15 {
			k = "15"
			val = avg[0]
			avgs[i].FifteenS = avg[0]
		}

		if val > maxes[k] {
			maxes[k] = val
		}
	}

	// Save CSV to file
	err := gocsv.MarshalFile(&avgs, csvFile)
	if err != nil {
		log.Fatalln(err)
	}
	kg := viper.GetFloat64("kg")

	fmt.Printf("Results from %d Power records:\n", len(powerArray))
	for i, sess := range activity.Sessions {
		fmt.Printf("Session results for session %d: \n", i)
		fmt.Printf("  Power: Avg, Max [%d, %d], wkg: [%.2f, %.2f]\n", sess.AvgPower, sess.MaxPower, float64(sess.AvgPower)/kg, float64(sess.MaxPower)/kg)
		fmt.Printf("  HR: Avg, Max [%d, %d]\n", sess.AvgHeartRate, sess.MaxHeartRate)
	}
	fmt.Printf("Max Pwr        \t\t20m: %.2f, \t10m: %.2f, \t5m: %.2f, \t1m: %.2f, \t30s: %.2f, \t15s: %.2f\n", maxes["1200"], maxes["600"], maxes["300"], maxes["60"], maxes["30"], maxes["15"])
	fmt.Printf("Max wkg (%.2f kg) \t20m: %.1f, \t10m: %.1f, \t5m: %.1f, \t1m: %.1f, \t30s: %.1f, \t15s: %.1f\n", kg, maxes["1200"]/kg, maxes["600"]/kg, maxes["300"]/kg, maxes["60"]/kg, maxes["30"]/kg, maxes["15"]/kg)
	fmt.Printf("FTP (0.95 * 20m) \tavg: %.1f\n", .95*(maxes["1200"]/kg))

}
