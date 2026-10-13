package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// SaveEarthworkCSV 按桩号输出土方汇总 CSV
func SaveEarthworkCSV(sections []SectionResult, path string) error {
	var sb strings.Builder
	sb.WriteString("桩号,间距(m),填方面积(㎡),挖方面积(㎡),清表面积(㎡)," +
		"平均填方面积(㎡),平均挖方面积(㎡),填方体积(m³),挖方体积(m³)," +
		"累计填方(m³),累计挖方(m³)\n")

	cumFill := 0.0
	cumCut := 0.0
	prevK := 0.0
	prevFill := 0.0
	prevCut := 0.0
	first := true

	for _, s := range sections {
		k := s.Station
		fill := s.FillArea
		cut := s.CutArea
		clear := s.ClearArea

		span, avgFill, avgCut, vFill, vCut := 0.0, 0.0, 0.0, 0.0, 0.0
		if !first {
			span = k - prevK
			avgFill = (prevFill + fill) / 2.0
			avgCut = (prevCut + cut) / 2.0
			vFill = avgFill * span
			vCut = avgCut * span
			cumFill += vFill
			cumCut += vCut
		}

		sb.WriteString(escapeCSV(HuaNum2K(k)))
		sb.WriteByte(',')
		sb.WriteString(f3(span))
		sb.WriteByte(',')
		sb.WriteString(f3(fill))
		sb.WriteByte(',')
		sb.WriteString(f3(cut))
		sb.WriteByte(',')
		sb.WriteString(f3(clear))
		sb.WriteByte(',')
		sb.WriteString(f3(avgFill))
		sb.WriteByte(',')
		sb.WriteString(f3(avgCut))
		sb.WriteByte(',')
		sb.WriteString(f3(vFill))
		sb.WriteByte(',')
		sb.WriteString(f3(vCut))
		sb.WriteByte(',')
		sb.WriteString(f3(cumFill))
		sb.WriteByte(',')
		sb.WriteString(f3(cumCut))
		sb.WriteByte('\n')

		prevK, prevFill, prevCut = k, fill, cut
		first = false
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("合计,,,,,,,,,%s,%s\n", f3(cumFill), f3(cumCut)))

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func f3(v float64) string {
	return strconv.FormatFloat(v, 'f', 3, 64)
}

func escapeCSV(s string) string {
	if !strings.ContainsAny(s, ",\"\n") {
		return s
	}
	return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
}
