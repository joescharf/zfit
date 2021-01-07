package zfit

import "fmt"

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
