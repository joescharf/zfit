package zfit

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"github.com/tormoder/fit"
)

type Zfit struct {
	Data    Data
	Results Results
	FitData *fit.File
	fileArg string
	records []*fit.RecordMsg
}

func New(fileArg string) *Zfit {
	// process argument (filename)
	dir, base := filepath.Split(fileArg)
	ext := filepath.Ext(base)
	fileName := strings.Replace(base, ext, "", -1)
	fitFileName := filepath.Join(dir, base)
	csvFileName := filepath.Join(dir, fmt.Sprintf("%s.csv", fileName))

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

	// Decode the fitData file data
	fitData, err := fit.Decode(bytes.NewReader(fitFile))
	if err != nil {
		log.Fatalln(err)
	}

	// Get the activity
	activity, err := fitData.Activity()
	if err != nil {
		log.Fatalln(err)
	}

	// Set some data:
	zfit := &Zfit{
		fileArg: fileArg,
		FitData: fitData,
		Data: Data{
			ProcessedAt: time.Now(),
			RecordedAt:  fitData.FileId.TimeCreated,
			FitFile:     fitFileName,
		},
		Results: Results{
			ProcessedAt: time.Now(),
			RecordedAt:  fitData.FileId.TimeCreated,
			FitFile:     fitFileName,
		},
	}

	// Set the records:
	zfit.records = activity.Records[0:]
	numRecords := len(activity.Records)

	// 1. INITIALIZE Data and Results Structs:
	zfit.Data.Timestamp = make([]time.Time, numRecords)
	zfit.Data.Power = make([]uint16, numRecords)
	zfit.Data.HR = make([]uint8, numRecords)
	zfit.Data.Speed = make([]uint16, numRecords)
	zfit.Data.Cadence = make([]uint8, numRecords)
	zfit.Data.MAArray = make([][]float64, numRecords)
	zfit.Data.Ticks = make([]*Tick, numRecords)

	zfit.SetBasicStats()

	return zfit
}

func (z *Zfit) SetWeight(kg float64) {
	z.Data.KG = kg
	z.Results.KG = kg
}

func (z *Zfit) SetBasicStats() {
	// 2 LOAD the DATA Struct and initial RESULTS:
	var (
		sumPower   uint32
		sumHR      uint32
		sumSpeed   uint32
		sumCadence uint32
	)

	numRecords := len(z.records)

	for i, v := range z.records {
		z.Data.Timestamp[i] = v.Timestamp

		z.Data.Power[i] = v.Power
		power64 := float64(v.Power)
		if power64 > z.Results.Power.Max {
			z.Results.Power.Max = power64
		}
		sumPower += uint32(v.Power)

		z.Data.HR[i] = v.HeartRate
		heart64 := float64(v.HeartRate)
		if heart64 > z.Results.HR.Max {
			z.Results.HR.Max = heart64
		}
		sumHR += uint32(v.HeartRate)

		z.Data.Speed[i] = v.Speed
		speed64 := float64(v.Speed)
		if speed64 > z.Results.Speed.Max {
			z.Results.Speed.Max = speed64
		}
		sumSpeed += uint32(v.Speed)

		z.Data.Cadence[i] = v.Cadence
		cadence64 := float64(v.Cadence)
		if cadence64 > z.Results.Cadence.Max {
			z.Results.Cadence.Max = cadence64
		}
		sumCadence += uint32(v.Cadence)
	}

	// RESULTS
	z.Results.Power.Label = "Power"
	z.Results.Power.Avg = float64(sumPower) / float64(numRecords)
	z.Results.HR.Label = "HR"
	z.Results.HR.Avg = float64(sumHR) / float64(numRecords)
	z.Results.Speed.Label = "Speed"
	z.Results.Speed.Avg = float64(sumSpeed) / float64(numRecords)
	z.Results.Cadence.Label = "Cadence"
	z.Results.Cadence.Avg = float64(sumCadence) / float64(numRecords)

	z.Missing()
	z.Timing()
}

func (z *Zfit) Missing() {
	for i, v := range z.records[1:] {
		diff := v.Timestamp.Sub(z.records[i].Timestamp)

		delta := int(diff / time.Second)
		if delta > 1 {
			fmt.Printf("Missing %v at %d\n", diff, i)
			z.Results.MissingRecs += delta
		}
	}
}

func (z *Zfit) Timing() {
	a, _ := z.FitData.Activity()
	spew.Dump(a.Sessions)
	sess := a.Sessions[0]
	z.Results.TotalElapsedTime = time.Millisecond * time.Duration(sess.TotalElapsedTime)
	z.Results.TotalTimerTime = time.Millisecond * time.Duration(sess.TotalTimerTime)
	// for i, v := range z.records[0:] {
	// }
}
func (z *Zfit) PrintMetaData() {

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Key", "Value"})
	data := [][]string{
		{"FitFile", fmt.Sprintf("%s", z.Results.FitFile)},
		{"Recorded On", fmt.Sprintf("%s", z.Results.RecordedAt.Format("2006-Jan-02 @ 15:04:05Z"))},
		{"Missing Recs", fmt.Sprintf("%d", z.Results.MissingRecs)},
		{"Total Elapsed", fmt.Sprintf("%v", z.Results.TotalElapsedTime)},
		{"Total Timer", fmt.Sprintf("%v", z.Results.TotalTimerTime)},
	}
	for _, v := range data {
		table.Append(v)
	}
	fmt.Printf("\nBasic Info for %s\n", z.Results.FitFile)
	table.Render()

}

func (z *Zfit) PrintBasicStats() {

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Metric", "Avg", "Max"})
	data := [][]string{
		{"Power", fmt.Sprintf("%.2f", z.Results.Power.Avg), fmt.Sprintf("%.2f", z.Results.Power.Max)},
		{"HR", fmt.Sprintf("%.2f", z.Results.HR.Avg), fmt.Sprintf("%.2f", z.Results.HR.Max)},
		{"Speed", fmt.Sprintf("%.2f", z.Results.Speed.Avg/1000), fmt.Sprintf("%.2f", z.Results.Speed.Max/1000)},
		{"Cadence", fmt.Sprintf("%.2f", z.Results.Cadence.Avg), fmt.Sprintf("%.2f", z.Results.Cadence.Max)},
	}
	for _, v := range data {
		table.Append(v)
	}
	fmt.Printf("\nBasic Stats for %s recorded on %v\n", z.Results.FitFile, z.Results.RecordedAt.Format("2006-Jan-02 @ 15:04:05Z"))
	table.Render()

}
