package zfit

import (
	"fmt"
	"time"
)

type Tick struct {
	Timestamp time.Time `csv:"timestamp"`
	T         int       `csv:"t"`
	Power     uint16    `csv:"power"`
	HR        uint8     `csv:"hr"`
	Speed     uint16    `csv:"speed"`
	Cadence   uint8     `csv:"cadence"`
	S1        float64   `csv:"1s"`
	S5        float64   `csv:"5s"`
	S15       float64   `csv:"15s"`
	S30       float64   `csv:"30s"`
	S60       float64   `csv:"1m"`
	S300      float64   `csv:"5m"`
	S600      float64   `csv:"10m"`
	S1200     float64   `csv:"20m"`
}

type Data struct {
	ProcessedAt time.Time
	RecordedAt  time.Time
	FitFile     string
	KG          float64
	Timestamp   []time.Time
	Power       []uint16
	HR          []uint8
	Speed       []uint16
	Cadence     []uint8
	MAArray     [][]float64
	MAMap       map[int][]float64
	Ticks       []*Tick
}
type Results struct {
	ProcessedAt time.Time
	RecordedAt  time.Time
	FitFile     string
	KG          float64
	Power       Stat
	HR          Stat
	Cadence     Stat
	Speed       Stat
	MAMax       map[int]*MAResult
}
type MAResult struct {
	Interval int
	KG       float64
	Avg      float64
	Max      float64
	MaxWkg   float64
}

func (r *MAResult) String() string {
	label := ""
	str := ""
	switch r.Interval {
	case 1:
		label = " 1s :"
	case 5:
		label = " 5s :"
	case 15:
		label = "15s :"
	case 30:
		label = "30s :"
	case 60:
		label = " 1m :"
	case 300:
		label = " 5m :"
	case 600:
		label = "10m :"
	case 1200:
		label = "20m :"
	}

	if r.MaxWkg > 0.0 {
		str = fmt.Sprintf("%s  Avg: %.2f, Max: %.2f, (%.2f wkg)", label, r.Avg, r.Max, r.MaxWkg)
	} else {
		str = fmt.Sprintf("%s  Avg: %.2f, Max: %.2f", label, r.Avg, r.Max)
	}
	return str
}

type Stat struct {
	Label string
	Avg   float64
	Max   float64
}

func (r *Stat) String() string {
	return fmt.Sprintf("%s: Avg: %.2f, Max: %.2f", r.Label, r.Avg, r.Max)
}

func IntervalToString(interval int) string {
	label := ""
	switch interval {
	case 1:
		label = "1s"
	case 5:
		label = "5s"
	case 15:
		label = "15s"
	case 30:
		label = "30s"
	case 60:
		label = "1m"
	case 300:
		label = "5m"
	case 600:
		label = "10m"
	case 1200:
		label = "20m"
	default:
		label = fmt.Sprintf("%d", interval)
	}
	return label
}
