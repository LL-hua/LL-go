package main

// DuanMianShuJu 断面数据：一个桩号 + 若干板块（宽度, 横坡%）
type DuanMianShuJu struct {
	ZhuangHao   float64
	BanKuaiJiHe [][2]float64
}

// LuJiCheDaoJieGuoBao 路基车道结果包
type LuJiCheDaoJieGuoBao struct {
	ZuoCeCheDaoJueDui [][2]float64
	YouCeCheDaoJueDui [][2]float64
	ZuoBanKuai        [][2]float64
	YouBanKuai        [][2]float64
}

// BianPoDuanLuo 边坡段落
type BianPoDuanLuo struct {
	QiShiZhuangHao  float64
	JieShuZhuangHao float64
	ZuoTian         [][2]float64
	ZuoWa           [][2]float64
	YouTian         [][2]float64
	YouWa           [][2]float64
}

// BianPoHouXuanBao 边坡候选包
type BianPoHouXuanBao struct {
	ZuoTianJueDui [][2]float64
	ZuoWaJueDui   [][2]float64
	YouTianJueDui [][2]float64
	YouWaJueDui   [][2]float64
}

// JiegoucengConfig 结构层配置
type JiegoucengConfig struct {
	StartStation   float64
	EndStation     float64
	LayerIndex     int
	LayerName      string
	Thickness      float64
	InnerStepWidth float64
	InnerSlope     float64
	OuterStepWidth float64
	OuterSlope     float64
}

type LeftJiegoucengConfig = JiegoucengConfig
type RightJiegoucengConfig = JiegoucengConfig

// SectionResult 单桩号计算结果
type SectionResult struct {
	Station float64
	CenterY float64

	LOuter [2]float64
	ROuter [2]float64
	LToe   [2]float64
	RToe   [2]float64

	Ground        [][2]float64
	Cleared       [][2]float64
	Design        [][2]float64
	LeftSubgrade  [][2]float64
	RightSubgrade [][2]float64
	Layers        [][][2]float64

	LeftSlabs  [][2]float64
	RightSlabs [][2]float64

	LeftSlopeRaw      [][2]float64
	LeftSlopeTrimmed  [][2]float64
	RightSlopeRaw     [][2]float64
	RightSlopeTrimmed [][2]float64

	FillArea   float64
	CutArea    float64
	ClearArea  float64
	LayerAreas []string

	Bounds [4]float64
}

// ComputeResult 计算结果包装
type ComputeResult struct {
	Success      bool
	Result       *SectionResult
	ErrorMessage string
}

func NewComputeOK(s *SectionResult) ComputeResult {
	return ComputeResult{Success: true, Result: s}
}

func NewComputeErr(msg string) ComputeResult {
	return ComputeResult{Success: false, ErrorMessage: msg}
}
