package main

import (
	"fmt"
	"math"
)

// SectionCalculator 单桩号断面计算核心
type SectionCalculator struct {
	Mesh             [][3]float64
	Pqx              [][]float64
	Sqx              [][]float64
	LeftWidths       []DuanMianShuJu
	RightWidths      []DuanMianShuJu
	SlopeData        []BianPoDuanLuo
	LeftCrossfalls   [][]float64
	RightCrossfalls  [][]float64
	LeftStructures   []LeftJiegoucengConfig
	RightStructures  []RightJiegoucengConfig
	ClearDepth       float64
}

func NewSectionCalculator(
	mesh [][3]float64, pqx, sqx [][]float64,
	leftWidths, rightWidths []DuanMianShuJu,
	slopeData []BianPoDuanLuo,
	leftCrossfalls, rightCrossfalls [][]float64,
	leftStructures []LeftJiegoucengConfig,
	rightStructures []RightJiegoucengConfig,
	clearDepth float64) *SectionCalculator {
	return &SectionCalculator{
		Mesh: mesh, Pqx: pqx, Sqx: sqx,
		LeftWidths: leftWidths, RightWidths: rightWidths,
		SlopeData: slopeData,
		LeftCrossfalls: leftCrossfalls, RightCrossfalls: rightCrossfalls,
		LeftStructures: leftStructures, RightStructures: rightStructures,
		ClearDepth: clearDepth,
	}
}

// Compute 计算单桩号断面
func (c *SectionCalculator) Compute(station float64) ComputeResult {
	centerY := HuaH(c.Sqx, station)

	dao := LuJiYaoSuYinQing_GetCrossectOnXY(station, centerY, c.LeftWidths, c.RightWidths)
	zuoDian := dao.ZuoCeCheDaoJueDui
	youDian := dao.YouCeCheDaoJueDui

	if len(zuoDian) == 0 || len(youDian) == 0 {
		return NewComputeErr("路基宽度数据缺失")
	}

	lOuterX := zuoDian[0][0]
	lOuterY := zuoDian[0][1]
	nYou := len(youDian)
	rOuterX := youDian[nYou-1][0]
	rOuterY := youDian[nYou-1][1]

	ground := GetMatchedBZ(c.Mesh, station, 3.0)
	if len(ground) == 0 {
		return NewComputeErr("地面线为空")
	}

	cleared := HuaOffsetPolyline(ground, c.ClearDepth)

	bpCfg := BianPoYinQing_K2BianPo(c.SlopeData, station)
	if bpCfg == nil {
		return NewComputeErr("边坡段落为空")
	}

	candidates := BianPoYinQing_GetAbsolute(bpCfg, lOuterX, lOuterY, rOuterX, rOuterY)
	lGroundY := FromXgetY(ground, lOuterX)
	var finalLeft [][2]float64
	if lOuterY > lGroundY {
		finalLeft = candidates.ZuoTianJueDui
	} else {
		finalLeft = candidates.ZuoWaJueDui
	}
	rGroundY := FromXgetY(ground, rOuterX)
	var finalRight [][2]float64
	if rOuterY > rGroundY {
		finalRight = candidates.YouTianJueDui
	} else {
		finalRight = candidates.YouWaJueDui
	}

	// 拼设计线（未裁剪）
	rawDesign := make([][2]float64, 0)
	for _, p := range finalLeft {
		rawDesign = append(rawDesign, p)
	}
	for _, p := range zuoDian {
		rawDesign = append(rawDesign, p)
	}
	rawDesign = append(rawDesign, [2]float64{0.0, centerY})
	for _, p := range youDian {
		rawDesign = append(rawDesign, p)
	}
	for _, p := range finalRight {
		rawDesign = append(rawDesign, p)
	}

	designTop := rawDesign

	// 左结构层
	leftCrossfall := LumianSlopeManager_InterpolateCrossfall(c.LeftCrossfalls, station)
	leftCenter := [2]float64{0.0, centerY}
	leftPolygons, leftSubgrade := LeftJiegoucengManager_ComputeCoordinates(
		station, leftCenter, leftCrossfall, c.LeftStructures)

	if len(leftPolygons) == 0 {
		return NewComputeErr("左幅结构层段落为空")
	}

	leftInnerX := leftSubgrade[len(leftSubgrade)-1][0]
	newH := FromXgetY(designTop, leftInnerX)
	leftCenter = [2]float64{0.0, newH}
	leftPolygons, leftSubgrade = LeftJiegoucengManager_ComputeCoordinates(
		station, leftCenter, leftCrossfall, c.LeftStructures)

	// 右结构层
	rightCrossfall := LumianSlopeManager_InterpolateCrossfall(c.RightCrossfalls, station)
	rightCenter := [2]float64{0.0, centerY}
	rightPolygons, rightSubgrade := RightJiegoucengManager_ComputeCoordinates(
		station, rightCenter, rightCrossfall, c.RightStructures)

	if len(rightPolygons) == 0 {
		return NewComputeErr("右幅结构层段落为空")
	}

	rightInnerX := rightSubgrade[0][0]
	newH = FromXgetY(designTop, rightInnerX)
	rightCenter = [2]float64{0.0, newH}
	rightPolygons, rightSubgrade = RightJiegoucengManager_ComputeCoordinates(
		station, rightCenter, rightCrossfall, c.RightStructures)

	// 拼最终设计线
	leftOuterAnchor := lOuterX
	if len(leftSubgrade) > 0 {
		leftOuterAnchor = leftSubgrade[0][0]
	}
	rightOuterAnchor := rOuterX
	if len(rightSubgrade) > 0 {
		rightOuterAnchor = rightSubgrade[len(rightSubgrade)-1][0]
	}

	designList := make([][2]float64, 0)
	designList = addOuterPart(rawDesign, designList, leftOuterAnchor, -1)
	for _, p := range leftSubgrade {
		designList = append(designList, p)
	}

	leftInnerAnchor := leftSubgrade[len(leftSubgrade)-1][0]
	rightInnerAnchor := rightSubgrade[0][0]
	for _, p := range rawDesign {
		if p[0] >= leftInnerAnchor && p[0] <= rightInnerAnchor {
			designList = append(designList, p)
		}
	}

	for _, p := range rightSubgrade {
		designList = append(designList, p)
	}
	designList = addOuterPart(rawDesign, designList, rightOuterAnchor, 1)

	designMat := designList

	// 算面积
	res := HuaCutAndFillArea(cleared, designMat, 8)
	fill := math.Abs(res[0])
	cut := math.Abs(res[1])
	minX, maxX, minY := res[2], res[3], res[4]
	clearPoly := BuildClearPolygon(ground, cleared, minX, maxX)
	clearArea := HuaPolygonArea(clearPoly)

	count := int(res[6])
	if count == 0 {
		return NewComputeErr("设计线与地面线无有效交点")
	}

	finalDesign := make([][2]float64, count)
	idx := 7
	for i := 0; i < count; i++ {
		finalDesign[i] = [2]float64{res[idx], res[idx+1]}
		idx += 2
	}

	// 结构层面积 + cleared1
	layerAreaTexts := make([]string, 0)
	layerPolygons := make([][][2]float64, 0)

	temp := make([][2]float64, 0)
	for _, p := range cleared {
		if p[0] >= minX && p[0] <= maxX {
			temp = append(temp, p)
		}
	}
	n := len(temp)
	cleared1 := make([][2]float64, n+2)
	cleared1[0] = finalDesign[0]
	for i := 0; i < n; i++ {
		cleared1[i+1] = temp[i]
	}
	cleared1[n+1] = finalDesign[len(finalDesign)-1]

	for i, poly := range leftPolygons {
		area := math.Abs(HuaPolygonArea(poly))
		name := fmt.Sprintf("L_Lay%d", i+1)
		if i < len(c.LeftStructures) && c.LeftStructures[i].LayerName != "" {
			name = c.LeftStructures[i].LayerName
		}
		layerAreaTexts = append(layerAreaTexts, fmt.Sprintf("L:%s:%.3f", name, area))
		layerPolygons = append(layerPolygons, poly)
	}
	for i, poly := range rightPolygons {
		area := math.Abs(HuaPolygonArea(poly))
		name := fmt.Sprintf("R_Lay%d", i+1)
		if i < len(c.RightStructures) && c.RightStructures[i].LayerName != "" {
			name = c.RightStructures[i].LayerName
		}
		layerAreaTexts = append(layerAreaTexts, fmt.Sprintf("R:%s:%.3f", name, area))
		layerPolygons = append(layerPolygons, poly)
	}

	// 边坡裁剪
	lToeX := finalDesign[0][0]
	lToeY := finalDesign[0][1]
	rToeX := finalDesign[len(finalDesign)-1][0]
	rToeY := finalDesign[len(finalDesign)-1][1]

	leftTrimmed := make([][2]float64, 0)
	leftTrimmed = append(leftTrimmed, [2]float64{lToeX, lToeY})
	for _, p := range finalLeft {
		if p[0] >= lToeX {
			leftTrimmed = append(leftTrimmed, p)
		}
	}
	leftTrimmed = append(leftTrimmed, [2]float64{lOuterX, lOuterY})

	rightTrimmed := make([][2]float64, 0)
	rightTrimmed = append(rightTrimmed, [2]float64{rOuterX, rOuterY})
	for _, p := range finalRight {
		if p[0] <= rToeX {
			rightTrimmed = append(rightTrimmed, p)
		}
	}
	rightTrimmed = append(rightTrimmed, [2]float64{rToeX, rToeY})

	// 组装
	sr := &SectionResult{
		Station:           station,
		CenterY:           centerY,
		LOuter:            [2]float64{lOuterX, lOuterY},
		ROuter:            [2]float64{rOuterX, rOuterY},
		LToe:              [2]float64{lToeX, lToeY},
		RToe:              [2]float64{rToeX, rToeY},
		Ground:            ground,
		Cleared:           cleared1,
		Design:            finalDesign,
		LeftSubgrade:      leftSubgrade,
		RightSubgrade:     rightSubgrade,
		Layers:            layerPolygons,
		LeftSlabs:         dao.ZuoBanKuai,
		RightSlabs:        dao.YouBanKuai,
		LeftSlopeRaw:      finalLeft,
		LeftSlopeTrimmed:  leftTrimmed,
		RightSlopeRaw:     finalRight,
		RightSlopeTrimmed: rightTrimmed,
		FillArea:          fill,
		CutArea:           cut,
		ClearArea:         clearArea,
		LayerAreas:        layerAreaTexts,
		Bounds:            [4]float64{minX, minY, maxX, res[5]},
	}

	return NewComputeOK(sr)
}

// addOuterPart 从 raw 里挑出 anchor 一侧的点加到 design，含插值锚点
func addOuterPart(raw, design [][2]float64, anchor float64, side int) [][2]float64 {
	for i := 0; i < len(raw); i++ {
		p := raw[i]
		ok := false
		if side < 0 {
			ok = p[0] <= anchor
		} else {
			ok = p[0] >= anchor
		}
		if ok {
			design = append(design, p)
		}

		if i > 0 {
			prev := raw[i-1]
			crossed := false
			if side < 0 && prev[0] < anchor && p[0] > anchor {
				crossed = true
			}
			if side > 0 && prev[0] > anchor && p[0] < anchor {
				crossed = true
			}
			if crossed {
				x1, y1 := prev[0], prev[1]
				x2, y2 := p[0], p[1]
				y := y1 + (y2-y1)*(anchor-x1)/(x2-x1)
				design = append(design, [2]float64{anchor, y})
			}
		}
	}
	return design
}
