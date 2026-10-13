package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	exeDir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取目录失败:", err)
		return
	}

	fmt.Println("程序目录:", exeDir)

	files := findXlsx(exeDir)
	if len(files) == 0 {
		fmt.Println("在", exeDir, "下找不到 .xlsx（除 test.xlsx）")
		fmt.Println("按回车退出...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	fmt.Println("找到", len(files), "个 xlsx：")
	for i, f := range files {
		fmt.Printf("  [%d] %s\n", i, filepath.Base(f))
	}
	fmt.Print("选择序号（默认 0）: ")
	reader := bufio.NewReader(os.Stdin)
	pickLine, _ := reader.ReadString('\n')
	pickLine = strings.TrimSpace(pickLine)
	pickIdx := 0
	if pickLine != "" {
		if v, err := strconv.Atoi(pickLine); err == nil {
			pickIdx = v
		}
	}
	if pickIdx < 0 || pickIdx >= len(files) {
		pickIdx = 0
	}

	xlsxPath := files[pickIdx]
	baseName := strings.TrimSuffix(filepath.Base(xlsxPath), filepath.Ext(xlsxPath))
	fmt.Println()
	fmt.Println("项目文件:", xlsxPath)
	fmt.Println("项目名称:", baseName)
	fmt.Println()

	sw := time.Now()
	data, err := LoadProject(xlsxPath)
	if err != nil {
		fmt.Println("加载失败:", err)
		fmt.Println("按回车退出...")
		reader.ReadString('\n')
		return
	}
	fmt.Printf("加载完成，耗时 %d ms\n", time.Since(sw).Milliseconds())
	fmt.Println("桩号总数:", len(data.Stations()))
	fmt.Println()

	for {
		fmt.Print("请输入桩号（米，例如 103200），或输入 q 退出，或输入 all 批量导出: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" || strings.EqualFold(input, "q") {
			break
		}

		if strings.EqualFold(input, "all") {
			doBatchExport(data, exeDir, baseName)
			continue
		}

		station, err := strconv.ParseFloat(input, 64)
		if err != nil {
			fmt.Println("桩号格式错误")
			continue
		}

		t := time.Now()
		r := data.Query(station)
		elapsed := time.Since(t)

		fmt.Println()
		fmt.Println("=========================")
		fmt.Printf(" 桩号: %s  (%g m)\n", HuaNum2K(station), station)

		if !r.Success {
			fmt.Println(" 失败:", r.ErrorMessage)
		} else {
			s := r.Result
			fmt.Printf(" 填方面积: %.3f ㎡\n", s.FillArea)
			fmt.Printf(" 挖方面积: %.3f ㎡\n", s.CutArea)
			fmt.Printf(" 清表面积: %.3f ㎡\n", s.ClearArea)
			fmt.Printf(" 中桩高程: %.3f\n", s.CenterY)
			fmt.Printf(" 清表范围: [%.3f, %.3f]\n", s.Bounds[0], s.Bounds[2])
			fmt.Println(" 结构层:")
			for _, txt := range s.LayerAreas {
				fmt.Println("   " + txt)
			}
		}
		fmt.Printf(" 耗时: %.2f ms\n", float64(elapsed.Microseconds())/1000.0)
		fmt.Println("=========================")
		fmt.Println()
	}

	fmt.Println("退出。")
}

func findXlsx(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	result := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".xlsx") {
			continue
		}
		if strings.EqualFold(name, "test.xlsx") {
			continue
		}
		result = append(result, filepath.Join(dir, name))
	}
	sort.Strings(result)
	return result
}

func doBatchExport(data *ProjectData, baseDir, baseName string) {
	stations := data.Stations()
	fmt.Println("共", len(stations), "个桩号，开始计算...")

	results := make([]SectionResult, 0, len(stations))
	errors := make([]string, 0)
	start := time.Now()
	okCount, failCount := 0, 0

	for i, k := range stations {
		rr := data.Query(k)
		if rr.Success && rr.Result != nil {
			results = append(results, *rr.Result)
			okCount++
		} else {
			failCount++
			msg := rr.ErrorMessage
			if msg == "" {
				msg = "未知错误"
			}
			errors = append(errors, HuaNum2K(k)+"\t"+msg)
		}

		if (i+1)%100 == 0 {
			fmt.Printf("  进度 %d/%d\n", i+1, len(stations))
		}
	}
	fmt.Printf("计算完成：成功 %d，失败 %d，耗时 %d ms\n",
		okCount, failCount, time.Since(start).Milliseconds())

	if len(results) == 0 {
		fmt.Println("没有可导出的断面。")
		return
	}

	// 输出目录：项目名 + result
	outDir := filepath.Join(baseDir, baseName+"result")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Println("创建输出目录失败:", err)
		return
	}
	fmt.Println("输出目录:", outDir)

	// 出图 HTML
	htmlPath := filepath.Join(outDir, baseName+"_出图.html")
	if err := SaveBatchHTML(results, baseName+" 横断面", htmlPath); err != nil {
		fmt.Println("导出出图失败:", err)
	} else {
		fi, _ := os.Stat(htmlPath)
		fmt.Println("已导出出图:", htmlPath)
		fmt.Printf("  文件大小: %d KB\n", fi.Size()/1024)
	}

	// 土方数据 CSV
	csvPath := filepath.Join(outDir, baseName+"_土方数据.csv")
	if err := SaveEarthworkCSV(results, csvPath); err != nil {
		fmt.Println("导出土方数据失败:", err)
	} else {
		fi, _ := os.Stat(csvPath)
		fmt.Println("已导出土方数据:", csvPath)
		fmt.Printf("  文件大小: %d KB\n", fi.Size()/1024)
	}

	// 出错位置 TXT
	if len(errors) > 0 {
		errPath := filepath.Join(outDir, baseName+"_出错位置.txt")
		var sb strings.Builder
		sb.WriteString(baseName + " 出错位置清单\n")
		sb.WriteString("生成时间: " + time.Now().Format("2006-01-02 15:04:05") + "\n")
		sb.WriteString(fmt.Sprintf("失败总数: %d\n\n", len(errors)))
		sb.WriteString("桩号\t原因\n")
		sb.WriteString("----------------------------------------\n")
		for _, line := range errors {
			sb.WriteString(line + "\n")
		}
		if err := os.WriteFile(errPath, []byte(sb.String()), 0644); err != nil {
			fmt.Println("导出出错位置失败:", err)
		} else {
			fmt.Println("已导出出错位置:", errPath)
			fmt.Println("  失败数:", len(errors))
		}
	} else {
		fmt.Println("没有失败桩号，未生成出错位置文件。")
	}
	fmt.Println()
}
