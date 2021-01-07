package zfit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocarina/gocsv"
	"github.com/mxmCherry/movavg"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
)

var INTERVALS = []int{1, 5, 15, 30, 60, 300, 600, 1200}

// CritPower computes critical power using a Multi Moving Average
// over the following intervals: {5s, 15s, 30s, 1m, 5m, 10m, 20m}
func (z *Zfit) CritPower() {

	RECORDS := z.records

	z.Results.MAMax = make(map[int]*MAResult)

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

	z.Data.MAMap = make(map[int][]float64)
	for _, v := range INTERVALS {
		z.Data.MAMap[v] = make([]float64, len(RECORDS))
	}

	// Iterate on power data and get the moving averages:
	for i, v := range RECORDS {

		power64 := float64(v.Power)
		avg := multiMA.Add(power64)

		z.Data.MAArray[i] = avg
		// Map the MA Array into the z.Data.MAMap. Include all values:
		z.Data.MAMap[1][i] = power64
		z.Data.MAMap[5][i] = avg[0]
		z.Data.MAMap[15][i] = avg[1]
		z.Data.MAMap[30][i] = avg[2]
		z.Data.MAMap[60][i] = avg[3]
		z.Data.MAMap[300][i] = avg[4]
		z.Data.MAMap[600][i] = avg[5]
		z.Data.MAMap[1200][i] = avg[6]

		// Map the avg array into Ticks struct:
		z.Data.Ticks[i] = &Tick{
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
			S1:        power64,
		}
	}

	kg := z.Data.KG

	// fmt.Printf("Results from %d Power RECORDS:\n", len(z.Data.Power))
	// for i, sess := range activity.Sessions {
	// 	fmt.Printf("Session results for session %d: \n", i)
	// 	fmt.Printf("  Power: Avg, Max [%d, %d], wkg: [%.2f, %.2f]\n", sess.AvgPower, sess.MaxPower, float64(sess.AvgPower)/kg, float64(sess.MaxPower)/kg)
	// 	fmt.Printf("  HR: Avg, Max [%d, %d]\n", sess.AvgHeartRate, sess.MaxHeartRate)
	// }

	for _, v := range INTERVALS {
		res := stats64R(v, z.Data.MAMap[v][v:], kg)
		z.Results.MAMax[v] = res
	}
}

func (z *Zfit) PrintCritPowerResults() {
	fmt.Printf("Critical Power Results, (%.2f kg)\n\n", z.Results.KG)

	for _, v := range INTERVALS {
		fmt.Println(z.Results.MAMax[v])
	}
}

func (z *Zfit) PrintCritPowerResultsTable() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Interval", "Average", "Max", "wkg"})

	for _, v := range INTERVALS {
		s := fmt.Sprintf("%s %.2f %.2f %.2f", IntervalToString(z.Results.MAMax[v].Interval), z.Results.MAMax[v].Avg, z.Results.MAMax[v].Max, z.Results.MAMax[v].MaxWkg)
		str := strings.Split(s, " ")
		table.Append(str)
	}

	fmt.Printf("\nCritical Power Results, (%.2f kg)\n", z.Results.KG)
	table.Render()

}
func (z *Zfit) CritPowerToCSV() {

	dir, base := filepath.Split(z.fileArg)
	ext := filepath.Ext(base)
	fileName := strings.Replace(base, ext, "", -1)
	csvFileName := filepath.Join(dir, fmt.Sprintf("%s.csv", fileName))

	log.WithFields(log.Fields{"csvFileName": csvFileName}).Debugln("Saving CritPower to CSV:")

	csvFile, err := os.OpenFile(csvFileName, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	defer csvFile.Close()

	err = gocsv.MarshalFile(&z.Data.Ticks, csvFile)
	if err != nil {
		log.Fatalln(err)
	}
}
