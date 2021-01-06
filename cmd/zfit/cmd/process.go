package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gocarina/gocsv"
	"github.com/joescharf/zfit/v2"
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

var DATA zfit.Data
var RESULTS zfit.Results

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

	// Check the weightKG parameter. if Zero, pull the data from config file.
	if weightKG == 0.0 {
		DATA.KG = viper.GetFloat64("kg")
	} else {
		DATA.KG = weightKG
	}

	DATA.CreatedAt = time.Now()
	RESULTS.CreatedAt = time.Now()
	DATA.FitFile = fitFileName
	RESULTS.FitFile = fitFileName
	RESULTS.KG = DATA.KG
	/////////////

	fmt.Printf("FIT File: Type %s, Manufacturer: %s, Created: %v\n", fitData.Type(), fitData.FileId.Manufacturer, fitData.FileId.TimeCreated)
	fmt.Println("Sports:")
	// Print the sport of the Sessions
	for _, session := range activity.Sessions {
		fmt.Printf("  %s\n", session.Sport)
	}
	fmt.Printf("Activity data: Sessions: %v, RECORDS: %v, Events: %v, Hrvs: %v\n", len(activity.Sessions), len(activity.Records), len(activity.Events), len(activity.Hrvs))
	fmt.Printf("Activity timestamp resolution: Timestamp 0: %v, Timestamp 1: %v, diff: %v\n", activity.Records[0].Timestamp, activity.Records[1].Timestamp, activity.Records[1].Timestamp.Unix()-activity.Records[0].Timestamp.Unix())

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
	// Multi Moving average: 5s, 15s, 30s, 1m, 5m, 10m, 20m
	INTERVALS := []int{5, 15, 30, 60, 300, 600, 1200}
	RECORDS := activity.Records[0:]

	// 1. INITIALIZE Data and Results Structs:

	DATA.Timestamp = make([]time.Time, len(RECORDS))
	DATA.Power = make([]uint16, len(RECORDS))
	DATA.HR = make([]uint8, len(RECORDS))
	DATA.Speed = make([]uint16, len(RECORDS))
	DATA.Cadence = make([]uint8, len(RECORDS))
	DATA.MAArray = make([][]float64, len(RECORDS))
	DATA.Ticks = make([]*Tick, len(RECORDS))

	RESULTS.MAMax = make(map[int]*MAResult)

	multi := movavg.Multi{
		movavg.NewSMA(5),
		movavg.NewSMA(15),
		movavg.NewSMA(30),
		movavg.NewSMA(60),
		movavg.NewSMA(300),
		movavg.NewSMA(600),
		movavg.NewSMA(1200),
	}
	multiMA := movavg.MultiThreadSafe(multi)

	DATA.MAMap = make(map[int][]float64)
	for _, v := range INTERVALS {
		DATA.MAMap[v] = make([]float64, len(RECORDS))
	}

	// 2 LOAD the DATA Struct and initial RESULTS:
	var (
		sumPower   uint32
		sumHR      uint32
		sumSpeed   uint32
		sumCadence uint32
	)

	for i, v := range RECORDS {
		DATA.Timestamp[i] = v.Timestamp

		DATA.Power[i] = v.Power
		power64 := float64(v.Power)
		if power64 > RESULTS.Power.Max {
			RESULTS.Power.Max = power64
		}
		sumPower += uint32(v.Power)

		DATA.HR[i] = v.HeartRate
		heart64 := float64(v.HeartRate)
		if heart64 > RESULTS.HR.Max {
			RESULTS.HR.Max = heart64
		}
		sumHR += uint32(v.HeartRate)

		DATA.Speed[i] = v.Speed
		speed64 := float64(v.Speed)
		if speed64 > RESULTS.Speed.Max {
			RESULTS.Speed.Max = speed64
		}
		sumSpeed += uint32(v.Speed)

		DATA.Cadence[i] = v.Cadence
		cadence64 := float64(v.Cadence)
		if cadence64 > RESULTS.Cadence.Max {
			RESULTS.Cadence.Max = cadence64
		}
		sumCadence += uint32(v.Cadence)
	}

	// RESULTS
	RESULTS.Power.Label = "Power"
	RESULTS.Power.Avg = float64(sumPower) / float64(len(RECORDS))
	RESULTS.HR.Label = "HR"
	RESULTS.HR.Avg = float64(sumHR) / float64(len(RECORDS))
	RESULTS.Speed.Label = "Speed"
	RESULTS.Speed.Avg = float64(sumSpeed) / float64(len(RECORDS))
	RESULTS.Cadence.Label = "Cadence"
	RESULTS.Cadence.Avg = float64(sumCadence) / float64(len(RECORDS))

	// Iterate on power data and get the moving averages:
	for i, v := range RECORDS {

		avg := multiMA.Add(float64(v.Power))

		DATA.MAArray[i] = avg
		// Map the MA Array into the DATA.MAMap. Include all values:
		DATA.MAMap[5][i] = avg[0]
		DATA.MAMap[15][i] = avg[1]
		DATA.MAMap[30][i] = avg[2]
		DATA.MAMap[60][i] = avg[3]
		DATA.MAMap[300][i] = avg[4]
		DATA.MAMap[600][i] = avg[5]
		DATA.MAMap[1200][i] = avg[6]

		// Map the avg array into Ticks struct:
		DATA.Ticks[i] = &Tick{
			T:         i,
			Timestamp: v.Timestamp,
			Power:     v.Power,
			HR:        v.HeartRate,
			Speed:     v.Speed,
			Cadence:   v.Cadence,
			S1200:     avg[6],
			S600:      avg[5],
			S300:      avg[4],
			S60:       avg[3],
			S30:       avg[2],
			S15:       avg[1],
			S5:        avg[0],
		}
	}

	// Save CSV to file
	err := gocsv.MarshalFile(&DATA.Ticks, csvFile)
	if err != nil {
		log.Fatalln(err)
	}
	kg := DATA.KG

	fmt.Printf("Results from %d Power RECORDS:\n", len(DATA.Power))
	for i, sess := range activity.Sessions {
		fmt.Printf("Session results for session %d: \n", i)
		fmt.Printf("  Power: Avg, Max [%d, %d], wkg: [%.2f, %.2f]\n", sess.AvgPower, sess.MaxPower, float64(sess.AvgPower)/kg, float64(sess.MaxPower)/kg)
		fmt.Printf("  HR: Avg, Max [%d, %d]\n", sess.AvgHeartRate, sess.MaxHeartRate)
	}

	fmt.Printf("\nImproved stats, (%.2f kg)\n\n", kg)

	str := statsu16P(DATA.Power[0:], kg)
	fmt.Printf("1s   : %s\n", str)

	for _, v := range INTERVALS {
		res := stats64R(v, DATA.MAMap[v][v:], kg)
		RESULTS.MAMax[v] = res
		fmt.Println(res)
	}
	spew.Dump(RESULTS)
}

func statsu16P(data []uint16, kg float64) (str string) {
	_, max, _, _, avg := statsu16(data)
	str = fmt.Sprintf("Avg: %.2f, Max: %d.00, (%.2f wkg)", avg, max, float64(max)/kg)
	return
}
func statsu16(data []uint16) (min uint64, max uint64, count int, sum uint64, avg float64) {

	count = len(data)
	for _, v := range data {
		v64 := uint64(v)
		sum += v64
		if v64 < min {
			min = v64
		}
		if v64 > max {
			max = v64
		}
	}
	avg = float64(sum) / float64(count)
	return
}

func stats64R(interval int, data []float64, kg float64) *MAResult {
	_, max, _, _, avg := stats64(data)
	res := &MAResult{
		Interval: interval,
		KG:       kg,
		Avg:      avg,
		Max:      max,
		MaxWkg:   max / kg,
	}
	return res
}
func stats64P(data []float64, kg float64) (str string) {
	_, max, _, _, avg := stats64(data)
	str = fmt.Sprintf("Avg: %.2f, Max: %.2f, (%.2f wkg)", avg, max, max/kg)
	return
}
func stats64(data []float64) (min float64, max float64, count int, sum float64, avg float64) {
	count = len(data)
	for _, v := range data {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	avg = sum / float64(count)
	return
}
