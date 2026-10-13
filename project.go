package main

import (
	"math"
	"strconv"
	"strings"
)

// ============================================================
// ProjectData 项目数据 + 计算器
// ============================================================

type ProjectData struct {
	Config  *AppConfig
	Pqx     [][]float64
	Sqx     [][]float64
	Kzb     [][]float64
	Mesh    [][3]float64

	LeftWidths  []DuanMianShuJu
	RightWidths []DuanMianShuJu
	SlopeData   []BianPoDuanLuo

	LeftCrossfalls  [][]float64
	RightCrossfalls [][]float64

	LeftStructures  []LeftJiegoucengConfig
	RightStructures []RightJiegoucengConfig

	Calculator *SectionCalculator

	// 坐标系 / 控制点（画地图用）
	Projection    string
	LonCenter     float64
	IsNorthern    bool
	ControlLonLat [][2]float64
	ControlXy     [][2]float64
}

// Query 查询单桩号
func (p *ProjectData) Query(station float64) ComputeResult {
	return p.Calculator.Compute(station)
}

// Stations 返回所有桩号
func (p *ProjectData) Stations() []float64 {
	arr := make([]float64, len(p.Kzb))
	for i, r := range p.Kzb {
		arr[i] = r[0]
	}
	return arr
}

// ============================================================
// Load 读取 xlsx，构造 ProjectData
// ============================================================

func LoadProject(xlsxPath string) (*ProjectData, error) {
	xlsx, err := NewXlsxReader(xlsxPath)
	if err != nil {
		return nil, err
	}
	defer xlsx.Close()

	data := &ProjectData{
		Projection: "gauss",
	}

	// 配置
	data.Config, err = readConfig(xlsx)
	if err != nil {
		return nil, err
	}

	// 平曲线 / 竖曲线 / 桩号 / 原地面
	if data.Pqx, err = readMatrix(xlsx, "平曲线", 8); err != nil {
		return nil, err
	}
	if data.Sqx, err = readMatrix(xlsx, "竖曲线", 3); err != nil {
		return nil, err
	}
	if data.Kzb, err = readMatrix(xlsx, "桩号", 1); err != nil {
		return nil, err
	}
	xyz, err := readMatrix(xlsx, "原地面", 3)
	if err != nil {
		return nil, err
	}

	// 左右板块
	if data.LeftWidths, err = readWidths(xlsx, "左板块"); err != nil {
		return nil, err
	}
	if data.RightWidths, err = readWidths(xlsx, "右板块"); err != nil {
		return nil, err
	}

	// 边坡
	if data.SlopeData, err = readSlope(xlsx, "边坡"); err != nil {
		return nil, err
	}

	// 结构层
	if data.LeftStructures, err = readLeftStructures(xlsx, "左结构层"); err != nil {
		return nil, err
	}
	if data.RightStructures, err = readRightStructures(xlsx, "右结构层"); err != nil {
		return nil, err
	}

	// 横坡
	if data.LeftCrossfalls, err = readMatrix(xlsx, "左结构层横坡", 2); err != nil {
		return nil, err
	}
	if data.RightCrossfalls, err = readMatrix(xlsx, "右结构层横坡", 2); err != nil {
		return nil, err
	}

	// 投影配置 + 控制点（可选）
	readProjectionConfig(xlsx, data)
	readControlPoints(xlsx, data)

	// 批量反算 mesh
	data.Mesh = HuaFsBatch(data.Pqx, xyz)

	// 构造计算器
	data.Calculator = NewSectionCalculator(
		data.Mesh, data.Pqx, data.Sqx,
		data.LeftWidths, data.RightWidths, data.SlopeData,
		data.LeftCrossfalls, data.RightCrossfalls,
		data.LeftStructures, data.RightStructures,
		data.Config.ClearDepth,
	)

	return data, nil
}

// ============================================================
// 读投影配置（配置 sheet 里几个特殊 key）
// ============================================================

func readProjectionConfig(xlsx *XlsxReader, data *ProjectData) {
	rows, err := xlsx.ReadSheet("配置")
	if err != nil {
		return
	}
	for _, row := range rows {
		key := strings.TrimSpace(getString(row, 0))
		val := strings.TrimSpace(getString(row, 1))
		if key == "" {
			continue
		}
		switch key {
		case "投影方式":
			data.Projection = strings.ToLower(val)
		case "中央子午线":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				data.LonCenter = v
			}
		case "南北半球":
			data.IsNorthern = strings.Contains(val, "北")
		}
	}
}

// ============================================================
// 读控制点 sheet：名字, X, Y, Z, 经度, 纬度
// ============================================================

func readControlPoints(xlsx *XlsxReader, data *ProjectData) {
	rows, err := xlsx.ReadSheet("控制点")
	if err != nil {
		data.ControlLonLat = nil
		data.ControlXy = nil
		return
	}

	llList := make([][2]float64, 0)
	xyList := make([][2]float64, 0)

	for _, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		x := getDouble(row, 1)
		y := getDouble(row, 2)
		lon := getDouble(row, 4)
		lat := getDouble(row, 5)
		if x == 0 && y == 0 {
			continue
		}
		xyList = append(xyList, [2]float64{x, y})
		llList = append(llList, [2]float64{lon, lat})
	}
	data.ControlXy = xyList
	data.ControlLonLat = llList
}

// ============================================================
// 通用读取
// ============================================================

func readMatrix(xlsx *XlsxReader, sheetName string, cols int) ([][]float64, error) {
	rows, err := xlsx.ReadSheet(sheetName)
	if err != nil {
		return nil, err
	}
	list := make([][]float64, 0, len(rows))
	for _, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		d := make([]float64, cols)
		for j := 0; j < cols; j++ {
			d[j] = getDouble(row, j)
		}
		list = append(list, d)
	}
	return list, nil
}

func readWidths(xlsx *XlsxReader, sheetName string) ([]DuanMianShuJu, error) {
	rows, err := xlsx.ReadSheet(sheetName)
	if err != nil {
		return nil, err
	}
	list := make([]DuanMianShuJu, 0, len(rows))
	for _, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		d := DuanMianShuJu{
			ZhuangHao: getDouble(row, 0),
		}

		maxKey := 0
		for k := range row {
			if k > maxKey {
				maxKey = k
			}
		}

		for j := 1; j+1 <= maxKey; j += 2 {
			if isCellEmpty(row, j) && isCellEmpty(row, j+1) {
				break
			}
			d.BanKuaiJiHe = append(d.BanKuaiJiHe, [2]float64{
				getDouble(row, j),
				getDouble(row, j+1),
			})
		}
		list = append(list, d)
	}
	return list, nil
}

func readSlope(xlsx *XlsxReader, sheetName string) ([]BianPoDuanLuo, error) {
	rows, err := xlsx.ReadSheet(sheetName)
	if err != nil {
		return nil, err
	}
	result := make([]BianPoDuanLuo, 0)
	cur := -1 // 当前段落索引，-1 表示还没有

	for _, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		k1, ok1 := tryGetDouble(row, 0)
		k2, ok2 := tryGetDouble(row, 1)
		typ := strings.ToLower(strings.TrimSpace(getString(row, 2)))
		dispStr := getString(row, 3)

		if ok1 && ok2 {
			result = append(result, BianPoDuanLuo{
				QiShiZhuangHao:  k1,
				JieShuZhuangHao: k2,
			})
			cur = len(result) - 1
		}
		if cur < 0 {
			continue
		}

		disp := parseDisp(dispStr)
		switch typ {
		case "leftfill":
			result[cur].ZuoTian = disp
		case "leftcut":
			result[cur].ZuoWa = disp
		case "rightfill":
			result[cur].YouTian = disp
		case "rightcut":
			result[cur].YouWa = disp
		}
	}
	return result, nil
}

func parseDisp(s string) [][2]float64 {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	nums := make([]float64, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		v, err := strconv.ParseFloat(t, 64)
		if err == nil {
			nums = append(nums, v)
		}
	}
	n := len(nums) / 2
	m := make([][2]float64, n)
	for i := 0; i < n; i++ {
		m[i] = [2]float64{nums[i*2], nums[i*2+1]}
	}
	return m
}

func readLeftStructures(xlsx *XlsxReader, sheetName string) ([]LeftJiegoucengConfig, error) {
	rows, err := xlsx.ReadSheet(sheetName)
	if err != nil {
		return nil, err
	}
	list := make([]LeftJiegoucengConfig, 0)
	for _, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		list = append(list, LeftJiegoucengConfig{
			StartStation:   getDouble(row, 0),
			EndStation:     getDouble(row, 1),
			LayerIndex:     int(getDouble(row, 2)),
			LayerName:      strings.TrimSpace(getString(row, 3)),
			Thickness:      getDouble(row, 4),
			InnerStepWidth: getDouble(row, 5),
			InnerSlope:     getDouble(row, 6),
			OuterStepWidth: getDouble(row, 7),
			OuterSlope:     getDouble(row, 8),
		})
	}
	return list, nil
}

func readRightStructures(xlsx *XlsxReader, sheetName string) ([]RightJiegoucengConfig, error) {
	rows, err := xlsx.ReadSheet(sheetName)
	if err != nil {
		return nil, err
	}
	list := make([]RightJiegoucengConfig, 0)
	for _, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		list = append(list, RightJiegoucengConfig{
			StartStation:   getDouble(row, 0),
			EndStation:     getDouble(row, 1),
			LayerIndex:     int(getDouble(row, 2)),
			LayerName:      strings.TrimSpace(getString(row, 3)),
			Thickness:      getDouble(row, 4),
			InnerStepWidth: getDouble(row, 5),
			InnerSlope:     getDouble(row, 6),
			OuterStepWidth: getDouble(row, 7),
			OuterSlope:     getDouble(row, 8),
		})
	}
	return list, nil
}

func readConfig(xlsx *XlsxReader) (*AppConfig, error) {
	cfg := NewAppConfig()
	rows, err := xlsx.ReadSheet("配置")
	if err != nil {
		return cfg, nil
	}
	for _, row := range rows {
		key := strings.TrimSpace(getString(row, 0))
		val := strings.TrimSpace(getString(row, 1))
		if key == "" {
			continue
		}
		switch key {
		case "ClearDepth":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.ClearDepth = v
			}
		case "XyzRows":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.XyzRows = v
			}
		case "XyzCols":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.XyzCols = v
			}
		}
	}
	return cfg, nil
}

// ============================================================
// 取值工具
// ============================================================

func isEmptyRow(row map[int]string) bool {
	if len(row) == 0 {
		return true
	}
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

func isCellEmpty(row map[int]string, idx int) bool {
	v, ok := row[idx]
	return !ok || strings.TrimSpace(v) == ""
}

func getDouble(row map[int]string, idx int) float64 {
	v, ok := row[idx]
	if !ok {
		return 0
	}
	s := strings.TrimSpace(v)
	if s == "" {
		return 0
	}
	d, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return d
}

func tryGetDouble(row map[int]string, idx int) (float64, bool) {
	v, ok := row[idx]
	if !ok {
		return 0, false
	}
	s := strings.TrimSpace(v)
	if s == "" {
		return 0, false
	}
	d, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return d, true
}

func getString(row map[int]string, idx int) string {
	v, ok := row[idx]
	if !ok {
		return ""
	}
	return v
}

// ============================================================
// 保留一个 math 引用（避免 unused import 报错）
// ============================================================
var _ = math.Abs
