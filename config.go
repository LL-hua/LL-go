package main

// AppConfig 项目配置
type AppConfig struct {
	ClearDepth float64
	XyzRows    int
	XyzCols    int
}

const (
	DefaultClearDepth = -0.3
	DefaultXyzRows    = 35
	DefaultXyzCols    = 60
)

func NewAppConfig() *AppConfig {
	return &AppConfig{
		ClearDepth: DefaultClearDepth,
		XyzRows:    DefaultXyzRows,
		XyzCols:    DefaultXyzCols,
	}
}
