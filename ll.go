package main

import (
	"fmt"
	"math"
	"sort"
)

// ============================================================
// 1. 桩号 / 角度工具
// ============================================================

// HuaNum2K 米 → K 桩号字符串
func HuaNum2K(meters float64) string {
	km := int(math.Floor(meters / 1000))
	m := meters - float64(km)*1000
	m = math.Round(m*100) / 100
	if m == math.Floor(m) {
		return fmt.Sprintf("K%d+%03d", km, int(m))
	}
	return fmt.Sprintf("K%d+%06.2f", km, m)
}

// HuaDmsToRadians 度分秒（DDD.MMSS）→ 弧度
func HuaDmsToRadians(dms float64) float64 {
	degrees := int(dms)
	fractional := dms - float64(degrees)
	minutes := int(fractional * 100)
	seconds := (fractional*100 - float64(minutes)) * 100
	totalDegrees := float64(degrees) + float64(minutes)/60.0 + seconds/3600.0
	return totalDegrees * math.Pi / 180.0
}

// HuaRadiansToDMS 弧度 → 度分秒（DDD.MMSS）
func HuaRadiansToDMS(radians float64) float64 {
	degrees := radians * (180 / math.Pi)
	d := int(math.Floor(degrees))
	remaining := (degrees - float64(d)) * 60
	m := int(math.Floor(remaining))
	s := (remaining - float64(m)) * 60

	mm := fmt.Sprintf("%02d", m)
	ss := fmt.Sprintf("%04.1f", s)
	dfm := fmt.Sprintf("%d%s%s", d, mm, ss)

	var v float64
	fmt.Sscanf(dfm, "%f", &v)
	return math.Round(v) / 10000.0
}

// ============================================================
// 2. 方位角 + 距离
// ============================================================

// HuaFwj 方位角 + 距离，返回 [距离, 方位角]
func HuaFwj(x0, y0, x1, y1 float64) [2]float64 {
	x := x1 - x0
	y := y1 - y0
	cd := math.Sqrt(x*x + y*y)
	hd := math.Atan2(y, x)
	if hd < 0 {
		hd += 2 * math.Pi
	}
	return [2]float64{cd, hd}
}

// ============================================================
// 3. 线路计算：HuaZs（高斯积分）
// ============================================================

var gaussRr = [4]float64{0.1739274226, 0.3260725774, 0.3260725774, 0.1739274226}
var gaussVv = [4]float64{0.0694318442, 0.3300094782, 0.6699905218, 0.9305681558}

// HuaZs 线路坐标 + 方位角
func HuaZs(xyk, xyx, xyy, xyhd, xycd,
	xyqdr, xyzdr, xyzy, jsk,
	jsb, jd float64) [3]float64 {

	jd = HuaDmsToRadians(jd)
	w := jsk - xyk

	// 圆曲线
	if math.Abs(xyqdr-xyzdr) < 0.01 && xyqdr > 0 {
		a := xyhd + xyzy*math.Pi/2
		da := w / xyqdr * xyzy
		ta := xyhd + da
		if ta < 0 {
			ta += 2 * math.Pi
		}
		x := xyx + xyqdr*math.Cos(a) - xyqdr*math.Cos(a+da) +
			jsb*math.Cos(a+da-xyzy*math.Pi/2+jd)
		y := xyy + xyqdr*math.Sin(a) - xyqdr*math.Sin(a+da) +
			jsb*math.Sin(a+da-xyzy*math.Pi/2+jd)
		return [3]float64{x, y, ta}
	}

	// 直线
	if xyqdr < 0.01 && xyzdr < 0.01 && xyzy < 0.01 {
		x := xyx + w*math.Cos(xyhd) + jsb*math.Cos(xyhd+jd)
		y := xyy + w*math.Sin(xyhd) + jsb*math.Sin(xyhd+jd)
		return [3]float64{x, y, xyhd}
	}

	// 缓和曲线
	qr := xyqdr
	if qr < 0.001 {
		qr = 99999999
	}
	zr := xyzdr
	if zr < 0.001 {
		zr = 99999999
	}
	c := 1 / qr
	d := (qr - zr) / (2 * xycd * qr * zr)
	xs, ys := 0.0, 0.0
	for i := 0; i < 4; i++ {
		v := gaussVv[i]
		r := gaussRr[i]
		f := xyhd + xyzy*v*w*(c+v*w*d)
		xs += r * math.Cos(f)
		ys += r * math.Sin(f)
	}
	fhz3 := xyhd + xyzy*w*(c+w*d)
	fhz3 = math.Mod(fhz3, 2*math.Pi)
	if fhz3 < 0 {
		fhz3 += 2 * math.Pi
	}
	x2 := xyx + w*xs + jsb*math.Cos(fhz3+jd)
	y2 := xyy + w*ys + jsb*math.Sin(fhz3+jd)
	return [3]float64{x2, y2, fhz3}
}

// HuaDantiaoxianludange 按里程定位段落并计算坐标
func HuaDantiaoxianludange(pqx [][]float64, k, b, z float64) [3]float64 {
	for i := 0; i < len(pqx); i++ {
		if k >= pqx[i][0] && k <= pqx[i][0]+pqx[i][4] {
			rad := HuaDmsToRadians(pqx[i][3])
			r := HuaZs(pqx[i][0], pqx[i][1], pqx[i][2], rad,
				pqx[i][4], pqx[i][5], pqx[i][6], pqx[i][7],
				k, b, z)
			return [3]float64{
				math.Round(r[0]*1000) / 1000,
				math.Round(r[1]*1000) / 1000,
				r[2],
			}
		}
	}
	return [3]float64{0, 0, 0}
}

// ============================================================
// 4. 反算
// ============================================================

// HuaFs 单点反算：坐标 → 桩号 + 偏距
func HuaFs(pqx [][]float64, fsx, fsy float64) [2]float64 {
	p0 := HuaDantiaoxianludange(pqx, pqx[0][0], 0, 0)
	cx, cy, cAngle := p0[0], p0[1], p0[2]
	jljd := HuaFwj(cx, cy, fsx, fsy)
	k := pqx[0][0]
	cz := jljd[0] * math.Cos(jljd[1]-cAngle)
	pj := jljd[0] * math.Sin(jljd[1]-cAngle)

	hang := len(pqx) - 1
	qdlc := pqx[0][0]
	zdlc := pqx[hang][0] + pqx[hang][4]
	iter := 0

	for math.Abs(cz) > 0.01 && iter < 15 {
		k += cz
		iter++
		if k < qdlc {
			return [2]float64{-1, -1}
		}
		if k > zdlc {
			return [2]float64{-2, -2}
		}

		p0 = HuaDantiaoxianludange(pqx, k, 0, 0)
		cx, cy, cAngle = p0[0], p0[1], p0[2]
		jljd = HuaFwj(cx, cy, fsx, fsy)
		cz = jljd[0] * math.Cos(jljd[1]-cAngle)
		pj = jljd[0] * math.Sin(jljd[1]-cAngle)
	}

	return [2]float64{
		math.Round(k*1000) / 1000,
		math.Round(pj*1000) / 1000,
	}
}

// HuaFsBatch 批量反算：点云 → [桩号, 偏距, 高程]，按桩号升序，桩号相同按偏距升序
func HuaFsBatch(pqx [][]float64, xy [][]float64) [][3]float64 {
	if len(pqx) == 0 || len(xy) == 0 {
		return nil
	}

	segCount := len(pqx)
	startK := make([]float64, segCount)
	endK := make([]float64, segCount)
	startX := make([]float64, segCount)
	startY := make([]float64, segCount)
	angleRad := make([]float64, segCount)
	segLen := make([]float64, segCount)
	A := make([]float64, segCount)
	B := make([]float64, segCount)
	C := make([]float64, segCount)

	for i := 0; i < segCount; i++ {
		startK[i] = pqx[i][0]
		segLen[i] = pqx[i][4]
		endK[i] = startK[i] + segLen[i]
		startX[i] = pqx[i][1]
		startY[i] = pqx[i][2]
		angleRad[i] = HuaDmsToRadians(pqx[i][3])
		A[i] = pqx[i][5]
		B[i] = pqx[i][6]
		C[i] = pqx[i][7]
	}

	pointCount := len(xy)
	results := make([][3]float64, 0, pointCount)

	qdlc := startK[0]
	zdlc := endK[segCount-1]

	for i := 0; i < pointCount; i++ {
		fsx := xy[i][0]
		fsy := xy[i][1]
		fsz := xy[i][2]

		var row [3]float64

		p0 := HuaZs(startK[0], startX[0], startY[0], angleRad[0], segLen[0],
			A[0], B[0], C[0], startK[0], 0, 0)
		cx, cy, cAngle := p0[0], p0[1], p0[2]

		fw := HuaFwj(cx, cy, fsx, fsy)
		dist, az := fw[0], fw[1]
		k := startK[0]
		cz := dist * math.Cos(az-cAngle)
		pj := dist * math.Sin(az-cAngle)

		iter := 0
		failed := false

		for math.Abs(cz) > 0.01 && iter < 15 {
			k += cz
			iter++

			if k < qdlc {
				row = [3]float64{-1, -1, fsz}
				failed = true
				break
			}
			if k > zdlc {
				row = [3]float64{-2, -2, fsz}
				failed = true
				break
			}

			segIdx := 0
			for j := 0; j < segCount; j++ {
				if k >= startK[j] && k <= endK[j] {
					segIdx = j
					break
				}
			}

			p0 = HuaZs(startK[segIdx], startX[segIdx], startY[segIdx],
				angleRad[segIdx], segLen[segIdx],
				A[segIdx], B[segIdx], C[segIdx],
				k, 0, 0)
			cx, cy, cAngle = p0[0], p0[1], p0[2]

			fw = HuaFwj(cx, cy, fsx, fsy)
			dist, az = fw[0], fw[1]
			cz = dist * math.Cos(az-cAngle)
			pj = dist * math.Sin(az-cAngle)
		}

		if !failed {
			row = [3]float64{
				math.Round(k*1000) / 1000,
				math.Round(pj*1000) / 1000,
				fsz,
			}
		}
		results = append(results, row)
	}

	// 按桩号升序，桩号相同按偏距升序
	sortSlice3(results)

	return results
}

// ============================================================
// 5. 匹配 / 折线工具
// ============================================================

// GetMatchedBZ 按桩号取横断面：二分定位 + 插入排序
func GetMatchedBZ(data [][3]float64, target, tolerance float64) [][2]float64 {
	rows := len(data)
	if rows == 0 {
		return nil
	}

	lower := target - tolerance
	left, right := 0, rows
	for left < right {
		mid := (left + right) / 2
		if data[mid][0] < lower {
			left = mid + 1
		} else {
			right = mid
		}
	}

	upper := target + tolerance

	count := 0
	for i := left; i < rows && data[i][0] <= upper; i++ {
		count++
	}
	if count == 0 {
		return nil
	}

	result := make([][2]float64, count)
	n := 0
	for i := left; i < rows && data[i][0] <= upper; i++ {
		b := data[i][1]
		z := data[i][2]
		pos := n - 1
		for pos >= 0 && result[pos][0] > b {
			result[pos+1][0] = result[pos][0]
			result[pos+1][1] = result[pos][1]
			pos--
		}
		result[pos+1][0] = b
		result[pos+1][1] = z
		n++
	}
	return result
}

// FromXgetY 按 X 二分 + 线性插值求 Y
func FromXgetY(points [][2]float64, targetX float64) float64 {
	const tolerance = 1e-6
	n := len(points)
	if n == 0 {
		return 0.0
	}

	low, high, index := 0, n-1, -1
	for low <= high {
		mid := (low + high) / 2
		midX := points[mid][0]
		if midX == targetX {
			index = mid
			break
		}
		if midX < targetX {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	if index >= 0 {
		return points[index][1]
	}

	rightIndex := low
	leftIndex := rightIndex - 1

	if rightIndex >= n {
		return points[n-1][1]
	}
	if leftIndex < 0 {
		return points[0][1]
	}

	xLeft := points[leftIndex][0]
	xRight := points[rightIndex][0]
	if math.Abs(xRight-xLeft) < tolerance {
		return math.Max(points[leftIndex][1], points[rightIndex][1])
	}

	yLeft := points[leftIndex][1]
	yRight := points[rightIndex][1]
	return yLeft + (yRight-yLeft)*(targetX-xLeft)/(xRight-xLeft)
}

// FromYgetX 按 Y 求所有交点 X 坐标
func FromYgetX(points [][2]float64, targetY float64) []float64 {
	const tolerance = 1e-12
	result := make([]float64, 0)
	n := len(points)
	if n == 0 {
		return result
	}

	for i := 0; i < n-1; i++ {
		x1, y1 := points[i][0], points[i][1]
		x2, y2 := points[i+1][0], points[i+1][1]

		y1Below := y1 <= targetY+tolerance
		y1Above := y1 >= targetY-tolerance
		y2Below := y2 <= targetY+tolerance
		y2Above := y2 >= targetY-tolerance
		cross := (y1Below && y2Above) || (y1Above && y2Below)
		if !cross {
			continue
		}

		if math.Abs(y2-y1) < tolerance {
			if len(result) == 0 || math.Abs(result[len(result)-1]-x1) > tolerance {
				result = append(result, x1)
			}
			if i == n-2 && (len(result) == 0 ||
				math.Abs(result[len(result)-1]-x2) > tolerance) {
				result = append(result, x2)
			}
			continue
		}

		t := (targetY - y1) / (y2 - y1)
		xIntersect := x1 + t*(x2-x1)
		if len(result) == 0 || math.Abs(result[len(result)-1]-xIntersect) > tolerance {
			result = append(result, xIntersect)
		}
	}

	return result
}

// ============================================================
// 排序辅助
// ============================================================

func sortSlice3(a [][3]float64) {
	n := len(a)
	// 简单插入排序（点数不多时够快；点数多时用标准库 sort）
	// 用标准库 sort.Slice 更稳
	sortSlice3Impl(a, n)
}

// ============================================================
// 排序辅助
// ============================================================


func sortSlice3Impl(a [][3]float64, n int) {
	sort.Slice(a, func(i, j int) bool {
		if a[i][0] != a[j][0] {
			return a[i][0] < a[j][0]
		}
		return a[i][1] < a[j][1]
	})
}

// ============================================================
// 6. 清表偏移 / 多边形面积 / 填挖面积
// ============================================================

// HuaOffsetPolyline 清表偏移：折线整体法向平移
func HuaOffsetPolyline(points [][2]float64, offset float64) [][2]float64 {
	if points == nil {
		return nil
	}
	ptCount := len(points)
	if ptCount < 2 {
		out := make([][2]float64, ptCount)
		copy(out, points)
		return out
	}

	segmentCount := ptCount - 1
	segLines := make([][4]float64, segmentCount)
	validSeg := make([]bool, segmentCount)

	for i := 0; i < segmentCount; i++ {
		dx := points[i+1][0] - points[i][0]
		dy := points[i+1][1] - points[i][1]
		ln := math.Sqrt(dx*dx + dy*dy)
		if ln < 1e-8 {
			continue
		}
		validSeg[i] = true
		nx := -dy / ln
		ny := dx / ln
		segLines[i][0] = points[i][0] + nx*offset
		segLines[i][1] = points[i][1] + ny*offset
		segLines[i][2] = points[i+1][0] + nx*offset
		segLines[i][3] = points[i+1][1] + ny*offset
	}

	firstIdx := -1
	for i := 0; i < segmentCount; i++ {
		if validSeg[i] {
			firstIdx = i
			break
		}
	}
	if firstIdx == -1 {
		return [][2]float64{}
	}

	rawVertices := make([][2]float64, segmentCount+1)
	rawCount := 0
	rawVertices[rawCount][0] = segLines[firstIdx][0]
	rawVertices[rawCount][1] = segLines[firstIdx][1]
	rawCount++

	prev := firstIdx
	for i := firstIdx + 1; i < segmentCount; i++ {
		if !validSeg[i] {
			continue
		}

		x1, y1 := segLines[prev][0], segLines[prev][1]
		x2, y2 := segLines[prev][2], segLines[prev][3]
		x3, y3 := segLines[i][0], segLines[i][1]
		x4, y4 := segLines[i][2], segLines[i][3]
		denom := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)

		if math.Abs(denom) > 1e-8 {
			t1 := x1*y2 - y1*x2
			t2 := x3*y4 - y3*x4
			rawVertices[rawCount][0] = (t1*(x3-x4) - (x1-x2)*t2) / denom
			rawVertices[rawCount][1] = (t1*(y3-y4) - (y1-y2)*t2) / denom
		} else {
			rawVertices[rawCount][0] = (x2 + x3) / 2.0
			rawVertices[rawCount][1] = (y2 + y3) / 2.0
		}
		rawCount++
		prev = i
	}
	rawVertices[rawCount][0] = segLines[prev][2]
	rawVertices[rawCount][1] = segLines[prev][3]
	rawCount++

	cleanVertices := make([][2]float64, rawCount)
	cleanCount := 0
	if rawCount > 0 {
		cleanVertices[0][0] = rawVertices[0][0]
		cleanVertices[0][1] = rawVertices[0][1]
		cleanCount = 1
	}

	for i := 1; i < rawCount; i++ {
		cx := rawVertices[i][0]
		cy := rawVertices[i][1]

		for cleanCount > 1 {
			lastX := cleanVertices[cleanCount-1][0]
			if cx < lastX-1e-5 {
				cleanCount--
			} else {
				break
			}
		}

		topX := cleanVertices[cleanCount-1][0]
		topY := cleanVertices[cleanCount-1][1]
		ddx := cx - topX
		ddy := cy - topY
		dist := math.Sqrt(ddx*ddx + ddy*ddy)

		if dist > 1e-4 {
			if cx < topX+1e-4 && cleanCount > 1 {
				cleanVertices[cleanCount-1][0] = (topX + cx) / 2.0
				cleanVertices[cleanCount-1][1] = (topY + cy) / 2.0
			} else {
				cleanVertices[cleanCount][0] = cx
				cleanVertices[cleanCount][1] = cy
				cleanCount++
			}
		}
	}

	output := make([][2]float64, cleanCount)
	copy(output, cleanVertices[:cleanCount])
	return output
}

// BuildClearPolygon 构建清表面积多边形
func BuildClearPolygon(ground, cleared [][2]float64, minX, maxX float64) [][2]float64 {
	list := make([][2]float64, 0, len(ground)+len(cleared)+4)
	list = append(list, [2]float64{minX, FromXgetY(ground, minX)})
	for _, g := range ground {
		if g[0] > minX && g[0] < maxX {
			list = append(list, [2]float64{g[0], g[1]})
		}
	}
	list = append(list, [2]float64{maxX, FromXgetY(ground, maxX)})
	list = append(list, [2]float64{maxX, FromXgetY(cleared, maxX)})
	for i := len(cleared) - 1; i >= 0; i-- {
		if cleared[i][0] > minX && cleared[i][0] < maxX {
			list = append(list, [2]float64{cleared[i][0], cleared[i][1]})
		}
	}
	list = append(list, [2]float64{minX, FromXgetY(cleared, minX)})
	return list
}

// HuaPolygonArea 多边形面积（鞋带公式）
func HuaPolygonArea(poly [][2]float64) float64 {
	if len(poly) < 3 {
		return 0
	}
	area := 0.0
	n := len(poly)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += poly[i][0] * poly[j][1]
		area -= poly[i][1] * poly[j][0]
	}
	if area < 0 {
		area = -area
	}
	return area / 2.0
}

// HuaCutAndFillArea 核心填挖面积：设计线 vs 地面线求交 + 鞋带累加
// 返回: [fill, cut, minX, maxX, minY, maxY, count, (x,y)...]
func HuaCutAndFillArea(dmx, sjx [][2]float64, extendDist float64) []float64 {
	if extendDist > 0 {
		n := len(dmx)
		if n >= 2 {
			x1, y1 := dmx[0][0], dmx[0][1]
			x2, y2 := dmx[1][0], dmx[1][1]
			dx, dy := x1-x2, y1-y2
			ln := math.Sqrt(dx*dx + dy*dy)
			if ln > 1e-12 {
				dx /= ln
				dy /= ln
				dmx[0][0] = x1 + dx*extendDist
				dmx[0][1] = y1 + dy*extendDist
			}
			x1 = dmx[n-2][0]
			y1 = dmx[n-2][1]
			x2 = dmx[n-1][0]
			y2 = dmx[n-1][1]
			dx = x2 - x1
			dy = y2 - y1
			ln = math.Sqrt(dx*dx + dy*dy)
			if ln > 1e-12 {
				dx /= ln
				dy /= ln
				dmx[n-1][0] = x2 + dx*extendDist
				dmx[n-1][1] = y2 + dy*extendDist
			}
		}
	}

	type inter struct {
		x, y float64
		i, j int
	}
	xys := make([]inter, 0)
	fill := 0.0
	cut := 0.0
	lenA := len(sjx)
	lenB := len(dmx)

	for i := 0; i < lenA-1; i++ {
		x1, y1 := sjx[i][0], sjx[i][1]
		x2, y2 := sjx[i+1][0], sjx[i+1][1]
		for j := 0; j < lenB-1; j++ {
			x3, y3 := dmx[j][0], dmx[j][1]
			x4, y4 := dmx[j+1][0], dmx[j+1][1]
			denom := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
			if denom != 0 {
				t := ((x1-x3)*(y3-y4) - (y1-y3)*(x3-x4)) / denom
				u := ((x1-x3)*(y1-y2) - (y1-y3)*(x1-x2)) / denom
				if t >= 0 && t <= 1 && u >= 0 && u <= 1 {
					px := x1 + t*(x2-x1)
					py := y1 + t*(y2-y1)
					xys = append(xys, inter{px, py, i, j})
				}
			}
		}
	}

	minX, maxX, minY, maxY := 0.0, 0.0, 0.0, 0.0
	leftXysIdx, rightXysIdx := 0, 0

	if len(xys) > 0 {
		minX = xys[0].x
		maxX = xys[0].x
		minY = xys[0].y
		maxY = xys[0].y
		for i := 1; i < len(xys); i++ {
			x, y := xys[i].x, xys[i].y
			if x < minX {
				minX = x
				leftXysIdx = i
			} else if x > maxX {
				maxX = x
				rightXysIdx = i
			}
			if y < minY {
				minY = y
			} else if y > maxY {
				maxY = y
			}
		}
	}

	for idx := 0; idx < len(xys)-1; idx++ {
		xy0 := xys[idx]
		xy1 := xys[idx+1]
		i0, i1 := xy0.i, xy1.i
		j0, j1 := xy0.j, xy1.j

		px, py := xy0.x, xy0.y
		signedArea := 0.0

		for k := i0 + 1; k <= i1; k++ {
			nx := sjx[k][0]
			ny := sjx[k][1]
			signedArea += px*ny - py*nx
			px, py = nx, ny
		}
		signedArea += px*xy1.y - py*xy1.x
		px, py = xy1.x, xy1.y

		for k := j1; k > j0; k-- {
			nx := dmx[k][0]
			ny := dmx[k][1]
			signedArea += px*ny - py*nx
			px, py = nx, ny
		}
		signedArea += px*xy0.y - py*xy0.x

		if signedArea > 0 {
			cut += signedArea / 2.0
		} else {
			fill += signedArea / 2.0
		}
	}

	finalSjxList := make([][2]float64, 0)
	if len(xys) >= 2 {
		leftIntersection := xys[leftXysIdx]
		rightIntersection := xys[rightXysIdx]
		leftSjxSegIdx := leftIntersection.i
		rightSjxSegIdx := rightIntersection.i

		finalSjxList = append(finalSjxList, [2]float64{leftIntersection.x, leftIntersection.y})
		for k := leftSjxSegIdx + 1; k <= rightSjxSegIdx; k++ {
			finalSjxList = append(finalSjxList, [2]float64{sjx[k][0], sjx[k][1]})
		}
		finalSjxList = append(finalSjxList, [2]float64{rightIntersection.x, rightIntersection.y})
	} else {
		for k := 0; k < lenA; k++ {
			finalSjxList = append(finalSjxList, [2]float64{sjx[k][0], sjx[k][1]})
		}
	}

	// 组装结果
	result := make([]float64, 0, 7+len(finalSjxList)*2)
	result = append(result, math.Round(fill*10000)/10000.0)
	result = append(result, math.Round(cut*10000)/10000.0)
	result = append(result, minX, maxX, minY, maxY)
	result = append(result, float64(len(finalSjxList)))
	for _, pt := range finalSjxList {
		result = append(result, pt[0], pt[1])
	}
	return result
}

// ============================================================
// 7. 竖曲线 / 横坡
// ============================================================

// HuaH 设计竖曲线高程
func HuaH(sqx [][]float64, k float64) float64 {
	n := len(sqx)
	if n == 0 {
		panic("设计竖曲线 wrong")
	}
	firstZ := sqx[0][0]
	lastZ := sqx[n-1][0]
	if k < firstZ || k > lastZ {
		return -1
	}
	if n == 1 {
		if math.Abs(k-firstZ) < 1e-9 {
			return sqx[0][1]
		}
		return -1
	}

	idx := -1
	for i := 0; i < n-1; i++ {
		if k >= sqx[i][0] && k <= sqx[i+1][0] {
			idx = i
			break
		}
	}
	if idx == -1 {
		return -1
	}

	z1, h1 := sqx[idx][0], sqx[idx][1]
	z2, h2 := sqx[idx+1][0], sqx[idx+1][1]
	slope := (h2 - h1) / (z2 - z1)
	result := h1 + slope*(k-z1)

	points := []int{idx, idx + 1}
	for _, p := range points {
		if p <= 0 || p >= n-1 {
			continue
		}
		r := sqx[p][2]
		if r <= 0 {
			continue
		}
		zv, hv := sqx[p][0], sqx[p][1]
		zPrev, hPrev := sqx[p-1][0], sqx[p-1][1]
		i1 := (hv - hPrev) / (zv - zPrev)
		zNext, hNext := sqx[p+1][0], sqx[p+1][1]
		i2 := (hNext - hv) / (zNext - zv)
		omega := i2 - i1
		T := r * math.Abs(omega) / 2.0
		start, end := zv-T, zv+T
		if k >= start && k <= end {
			var tanElev float64
			if k <= zv {
				tanElev = hv + i1*(k-zv)
			} else {
				tanElev = hv + i2*(k-zv)
			}
			var x float64
			if k <= zv {
				x = k - start
			} else {
				x = end - k
			}
			var y float64
			if omega >= 0 {
				y = x * x / (2.0 * r)
			} else {
				y = -x * x / (2.0 * r)
			}
			result = tanElev + y
		}
	}
	return math.Round(result*1000) / 1000
}

// HuaSlope 横坡线性插值
func HuaSlope(mileage float64, points [][]float64) float64 {
	if points == nil {
		panic("points nil")
	}
	n := len(points)
	if n == 0 {
		panic("点表不能为空")
	}
	if len(points[0]) < 2 {
		panic("点表必须两列")
	}
	for i := 1; i < n; i++ {
		if points[i][0] <= points[i-1][0] {
			panic("里程必须严格递增")
		}
	}

	if mileage <= points[0][0] {
		return points[0][1]
	}
	if mileage >= points[n-1][0] {
		return points[n-1][1]
	}

	for i := 0; i < n-1; i++ {
		k1, k2 := points[i][0], points[i+1][0]
		if mileage >= k1 && mileage <= k2 {
			s1, s2 := points[i][1], points[i+1][1]
			if math.Abs(k2-k1) < 1e-9 {
				return s1
			}
			t := (mileage - k1) / (k2 - k1)
			return s1 + t*(s2-s1)
		}
	}
	return points[0][1]
}
