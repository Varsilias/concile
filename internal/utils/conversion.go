package utils

import "fmt"

const (
	_ = 1 << (10 * iota)
	KiB
	MiB
	GiB
)

func Bytes(size int) string {
	p := fmt.Sprintf
	if size >= GiB {
		gb := calculateRemainder(size, 30)
		return p("%.2f GiB", gb)
	}

	if size >= MiB {
		mb := calculateRemainder(size, 20)
		return p("%.2f MiB", mb)
	}

	if size >= KiB {
		kb := calculateRemainder(size, 10)
		return p("%.2f KiB", kb)
	}

	return p("%d B", size)

}

func calculateRemainder(size int, unit int) float64 {
	value := 1 << unit
	hb := size >> unit
	rem := size % value
	return float64(hb) + (float64(rem) / float64(value))
}
