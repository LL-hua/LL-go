package main

import (
	"math"
	"sort"
)

// ============================================================
// 1. 路基宽度引擎
// ============================================================

// LuJiYaoSuYinQing_GetCrossectOnXY 按桩号插值路基左右车道点
func LuJiYaoSuYinQing_GetCrossectOnXY(
	muBiaoZhuangHao, zhongZhuangGaoCheng float64,
	suoYouZuoData, suoYouYouData []DuanMianShuJu) LuJiCheDaoJieGuoBao {

	var jieGuoBao LuJiCheDaoJieGuoBao

	// 左侧
	if len(suoYouZuoData) > 0 {
		stBefore := suoYouZuoData[0].ZhuangHao
		stAfter := suoYouZuoData[len(suoYouZuoData)-1].ZhuangHao
		dmBefore := suoYouZuoData[0]
		dmAfter := suoYouZuoData[len(suoYouZuoData)-1]

		if muBiaoZhuangHao <= stBefore {
			dmAfter = dmBefore
			stAfter = stBefore
		} else if muBiaoZhuangHao >= stAfter {
			dmBefore = dmAfter
			stBefore = stAfter
		} else {
			for i := 1; i < len(suoYouZuoData); i++ {
				if suoYouZuoData[i].ZhuangHao >= muBiaoZhuangHao {
					stAfter = suoYouZuoData[i].ZhuangHao
					dmAfter = suoYouZuoData[i]
					stBefore = suoYouZuoData[i-1].ZhuangHao
					dmBefore = suoYouZuoData[i-1]
					break
				}
			}
		}

		ratio := 0.0
		if stBefore != stAfter {
			ratio = (muBiaoZhuangHao - stBefore) / (stAfter - stBefore)
		}
		curZuoX := 0.0
		curZuoY := zhongZhuangGaoCheng

		dianList := make([][2]float64, 0)
		banKuaiList := make([][2]float64, 0)

		n := len(dmBefore.BanKuaiJiHe)
		if len(dmAfter.BanKuaiJiHe) < n {
			n = len(dmAfter.BanKuaiJiHe)
		}
		for k := 0; k < n; k++ {
			wBefore := dmBefore.BanKuaiJiHe[k][0]
			sBefore := dmBefore.BanKuaiJiHe[k][1]
			wAfter := dmAfter.BanKuaiJiHe[k][0]
			sAfter := dmAfter.BanKuaiJiHe[k][1]

			curWidth := wBefore + ratio*(wAfter-wBefore)
			curSlopePercent := sBefore + ratio*(sAfter-sBefore)
			curSlopeVal := curSlopePercent / 100.0

			curZuoX -= curWidth
			curZuoY += curWidth * curSlopeVal

			dianList = append(dianList, [2]float64{curZuoX, curZuoY})
			banKuaiList = append(banKuaiList, [2]float64{curWidth, curSlopePercent})
		}

		reverse2(dianList)
		reverse2(banKuaiList)

		jieGuoBao.ZuoCeCheDaoJueDui = dianList
		jieGuoBao.ZuoBanKuai = banKuaiList
	} else {
		jieGuoBao.ZuoCeCheDaoJueDui = nil
		jieGuoBao.ZuoBanKuai = nil
	}

	// 右侧
	if len(suoYouYouData) > 0 {
		stBefore := suoYouYouData[0].ZhuangHao
		stAfter := suoYouYouData[len(suoYouYouData)-1].ZhuangHao
		dmBefore := suoYouYouData[0]
		dmAfter := suoYouYouData[len(suoYouYouData)-1]

		if muBiaoZhuangHao <= stBefore {
			dmAfter = dmBefore
			stAfter = stBefore
		} else if muBiaoZhuangHao >= stAfter {
			dmBefore = dmAfter
			stBefore = stAfter
		} else {
			for i := 1; i < len(suoYouYouData); i++ {
				if suoYouYouData[i].ZhuangHao >= muBiaoZhuangHao {
					stAfter = suoYouYouData[i].ZhuangHao
					dmAfter = suoYouYouData[i]
					stBefore = suoYouYouData[i-1].ZhuangHao
					dmBefore = suoYouYouData[i-1]
					break
				}
			}
		}

		ratio := 0.0
		if stBefore != stAfter {
			ratio = (muBiaoZhuangHao - stBefore) / (stAfter - stBefore)
		}
		curYouX := 0.0
		curYouY := zhongZhuangGaoCheng

		dianList := make([][2]float64, 0)
		banKuaiList := make([][2]float64, 0)

		n := len(dmBefore.BanKuaiJiHe)
		if len(dmAfter.BanKuaiJiHe) < n {
			n = len(dmAfter.BanKuaiJiHe)
		}
		for k := 0; k < n; k++ {
			wBefore := dmBefore.BanKuaiJiHe[k][0]
			sBefore := dmBefore.BanKuaiJiHe[k][1]
			wAfter := dmAfter.BanKuaiJiHe[k][0]
			sAfter := dmAfter.BanKuaiJiHe[k][1]

			curWidth := wBefore + ratio*(wAfter-wBefore)
			curSlopePercent := sBefore + ratio*(sAfter-sBefore)
			curSlopeVal := curSlopePercent / 100.0

			curYouX += curWidth
			curYouY += curWidth * curSlopeVal

			dianList = append(dianList, [2]float64{curYouX, curYouY})
			banKuaiList = append(banKuaiList, [2]float64{curWidth, curSlopePercent})
		}

		jieGuoBao.YouCeCheDaoJueDui = dianList
		jieGuoBao.YouBanKuai = banKuaiList
	} else {
		jieGuoBao.YouCeCheDaoJueDui = nil
		jieGuoBao.YouBanKuai = nil
	}

	return jieGuoBao
}

// ============================================================
// 2. 边坡引擎
// ============================================================

// BianPoYinQing_K2BianPo 按桩号定位边坡段落
func BianPoYinQing_K2BianPo(suoYouBianPo []BianPoDuanLuo, muBiaoZhuangHao float64) *BianPoDuanLuo {
	for i := range suoYouBianPo {
		d := &suoYouBianPo[i]
		if muBiaoZhuangHao >= d.QiShiZhuangHao && muBiaoZhuangHao < d.JieShuZhuangHao {
			return d
		}
	}
	return nil
}

// BianPoYinQing_GetAbsolute 边坡候选（累积到绝对坐标）
func BianPoYinQing_GetAbsolute(
	dangQianBianPo *BianPoDuanLuo,
	zuoYuanX, zuoYuanY, youYuanX, youYuanY float64) BianPoHouXuanBao {

	var houxuan BianPoHouXuanBao
	if dangQianBianPo == nil {
		return houxuan
	}

	houxuan.ZuoTianJueDui = accumulate(dangQianBianPo.ZuoTian, zuoYuanX, zuoYuanY, true)
	houxuan.ZuoWaJueDui = accumulate(dangQianBianPo.ZuoWa, zuoYuanX, zuoYuanY, true)
	houxuan.YouTianJueDui = accumulate(dangQianBianPo.YouTian, youYuanX, youYuanY, false)
	houxuan.YouWaJueDui = accumulate(dangQianBianPo.YouWa, youYuanX, youYuanY, false)

	return houxuan
}

func accumulate(steps [][2]float64, startX, startY float64, reverse bool) [][2]float64 {
	if len(steps) == 0 {
		return nil
	}
	n := len(steps)
	result := make([][2]float64, n)
	x, y := startX, startY
	for i := 0; i < n; i++ {
		x += steps[i][0]
		y += steps[i][1]
		result[i][0] = x
		result[i][1] = y
	}
	if reverse {
		reverse2(result)
	}
	return result
}

// ============================================================
// 3. 横坡引擎
// ============================================================

// LumianSlopeManager_InterpolateCrossfall 按里程插值横坡
func LumianSlopeManager_InterpolateCrossfall(records [][]float64, currentStation float64) float64 {
	if records == nil {
		return 0.0
	}
	n := len(records)
	if n == 0 {
		return 0.0
	}
	if n == 1 {
		return records[0][1]
	}

	if currentStation <= records[0][0] {
		return records[0][1]
	}
	if currentStation >= records[n-1][0] {
		return records[n-1][1]
	}

	low, high := 0, n-1
	for low <= high {
		mid := low + (high-low)/2
		midStation := records[mid][0]
		if math.Abs(midStation-currentStation) < 0.00001 {
			return records[mid][1]
		}
		if midStation < currentStation {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	s1, v1 := records[high][0], records[high][1]
	s2, v2 := records[low][0], records[low][1]
	if math.Abs(s2-s1) < 0.00001 {
		return v1
	}
	return v1 + (v2-v1)*(currentStation-s1)/(s2-s1)
}

// ============================================================
// 4. 左结构层引擎
// ============================================================

// LeftJiegoucengManager_ComputeCoordinates 左结构层坐标计算
// 返回 (polygons, subgradeLine)
func LeftJiegoucengManager_ComputeCoordinates(
	station float64, centerPoint [2]float64, crossfall float64,
	configs []LeftJiegoucengConfig) ([][][2]float64, [][2]float64) {

	polygons := make([][][2]float64, 0)
	activeLayers := make([]LeftJiegoucengConfig, 0)
	for _, cfg := range configs {
		if station >= cfg.StartStation && station <= cfg.EndStation {
			activeLayers = append(activeLayers, cfg)
		}
	}
	sort.Slice(activeLayers, func(i, j int) bool {
		return activeLayers[i].LayerIndex < activeLayers[j].LayerIndex
	})

	if len(activeLayers) == 0 {
		return polygons, nil
	}

	layerCount := len(activeLayers)
	k := -crossfall / 100.0

	topOutX := make([]float64, layerCount)
	topOutY := make([]float64, layerCount)
	botOutX := make([]float64, layerCount)
	botOutY := make([]float64, layerCount)
	topInX := make([]float64, layerCount)
	topInY := make([]float64, layerCount)
	botInX := make([]float64, layerCount)
	botInY := make([]float64, layerCount)

	firstLayer := activeLayers[0]
	curTopInX := centerPoint[0] + firstLayer.InnerStepWidth
	curTopInY := centerPoint[1]

	curTopOutX := curTopInX + firstLayer.OuterStepWidth
	curTopOutY := curTopInY + (curTopOutX-curTopInX)*k

	for i := 0; i < layerCount; i++ {
		cfg := activeLayers[i]

		if i > 0 {
			curTopOutX = botOutX[i-1] + cfg.OuterStepWidth
			curTopOutY = botOutY[i-1] + (curTopOutX-botOutX[i-1])*k
			curTopInX = botInX[i-1] + cfg.InnerStepWidth
			curTopInY = botInY[i-1] + (curTopInX-botInX[i-1])*k
		}

		topOutX[i] = curTopOutX
		topOutY[i] = curTopOutY
		topInX[i] = curTopInX
		topInY[i] = curTopInY

		outN := math.Abs(cfg.OuterSlope)
		dx := -(outN * cfg.Thickness) / (1.0 - outN*k)
		botOutXv := curTopOutX + dx
		botOutYv := curTopOutY - cfg.Thickness + dx*k
		botOutX[i] = botOutXv
		botOutY[i] = botOutYv

		botInX[i] = curTopInX
		botInY[i] = curTopInY - cfg.Thickness

		poly := make([][2]float64, 4)
		poly[0] = [2]float64{topOutX[i], topOutY[i]}
		poly[1] = [2]float64{botOutX[i], botOutY[i]}
		poly[2] = [2]float64{botInX[i], botInY[i]}
		poly[3] = [2]float64{topInX[i], topInY[i]}
		polygons = append(polygons, poly)
	}

	combined := make([][2]float64, 0)
	for i := 0; i < layerCount; i++ {
		combined = append(combined, [2]float64{topOutX[i], topOutY[i]})
		combined = append(combined, [2]float64{botOutX[i], botOutY[i]})
	}

	innerList := make([][2]float64, 0)
	innerList = append(innerList, [2]float64{topInX[0], centerPoint[1]})
	for i := 0; i < layerCount; i++ {
		innerList = append(innerList, [2]float64{topInX[i], topInY[i]})
		innerList = append(innerList, [2]float64{botInX[i], botInY[i]})
	}
	reverse2(innerList)
	combined = append(combined, innerList...)

	return polygons, combined
}

// ============================================================
// 5. 右结构层引擎
// ============================================================

// RightJiegoucengManager_ComputeCoordinates 右结构层坐标计算
func RightJiegoucengManager_ComputeCoordinates(
	station float64, centerPoint [2]float64, crossfall float64,
	configs []RightJiegoucengConfig) ([][][2]float64, [][2]float64) {

	polygons := make([][][2]float64, 0)
	activeLayers := make([]RightJiegoucengConfig, 0)
	for _, cfg := range configs {
		if station >= cfg.StartStation && station <= cfg.EndStation {
			activeLayers = append(activeLayers, cfg)
		}
	}
	sort.Slice(activeLayers, func(i, j int) bool {
		return activeLayers[i].LayerIndex < activeLayers[j].LayerIndex
	})

	if len(activeLayers) == 0 {
		return polygons, nil
	}

	layerCount := len(activeLayers)
	k := crossfall / 100.0

	topOutX := make([]float64, layerCount)
	topOutY := make([]float64, layerCount)
	botOutX := make([]float64, layerCount)
	botOutY := make([]float64, layerCount)
	topInX := make([]float64, layerCount)
	topInY := make([]float64, layerCount)
	botInX := make([]float64, layerCount)
	botInY := make([]float64, layerCount)

	firstLayer := activeLayers[0]
	curTopInX := centerPoint[0] + firstLayer.InnerStepWidth
	curTopInY := centerPoint[1]

	curTopOutX := curTopInX + firstLayer.OuterStepWidth
	curTopOutY := curTopInY + (curTopOutX-curTopInX)*k

	for i := 0; i < layerCount; i++ {
		cfg := activeLayers[i]

		if i > 0 {
			curTopOutX = botOutX[i-1] + cfg.OuterStepWidth
			curTopOutY = botOutY[i-1] + (curTopOutX-botOutX[i-1])*k
			curTopInX = botInX[i-1] + cfg.InnerStepWidth
			curTopInY = botInY[i-1] + (curTopInX-botInX[i-1])*k
		}

		topOutX[i] = curTopOutX
		topOutY[i] = curTopOutY
		topInX[i] = curTopInX
		topInY[i] = curTopInY

		outN := math.Abs(cfg.OuterSlope)
		dx := (outN * cfg.Thickness) / (1.0 + outN*k)
		botOutXv := curTopOutX + dx
		botOutYv := curTopOutY - cfg.Thickness + (botOutXv-curTopOutX)*k
		botOutX[i] = botOutXv
		botOutY[i] = botOutYv

		botInX[i] = curTopInX
		botInY[i] = curTopInY - cfg.Thickness

		poly := make([][2]float64, 4)
		poly[0] = [2]float64{topOutX[i], topOutY[i]}
		poly[1] = [2]float64{botOutX[i], botOutY[i]}
		poly[2] = [2]float64{botInX[i], botInY[i]}
		poly[3] = [2]float64{topInX[i], topInY[i]}
		polygons = append(polygons, poly)
	}

	combined := make([][2]float64, 0)
	combined = append(combined, [2]float64{topInX[0], centerPoint[1]})
	for i := 0; i < layerCount; i++ {
		combined = append(combined, [2]float64{topInX[i], topInY[i]})
		combined = append(combined, [2]float64{botInX[i], botInY[i]})
	}

	outerList := make([][2]float64, 0)
	for i := 0; i < layerCount; i++ {
		outerList = append(outerList, [2]float64{topOutX[i], topOutY[i]})
		outerList = append(outerList, [2]float64{botOutX[i], botOutY[i]})
	}
	reverse2(outerList)
	combined = append(combined, outerList...)

	return polygons, combined
}

// ============================================================
// 通用辅助
// ============================================================

func reverse2(a [][2]float64) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}
