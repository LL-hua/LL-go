# hdm-go

公路路基横断面计算工具 —— 读 Excel 项目数据，批量算出每个桩号的填挖面积、清表面积、结构层面积，导出交互式断面图和土方数据。

纯 Go 标准库，零第三方依赖，单文件可执行，**无需安装任何运行时**。

## 功能

- 读取 `.xlsx` 项目数据（平曲线、竖曲线、桩号、原地面、左右板块、边坡、左右结构层、横坡、控制点）
- 线路坐标正反算（直线 / 圆曲线 / 缓和曲线，高斯积分）
- 逐桩横断面计算：地面线、清表线、设计线、边坡裁剪、结构层
- 填方面积 / 挖方面积 / 清表面积（鞋带公式 + 设计线 vs 地面线求交）
- 批量导出：
  - `xxx_出图.html` —— 可翻页 / 缩放 / 拖动的交互式断面浏览器（手机浏览器直接打开）
  - `xxx_土方数据.csv` —— 平均断面法逐桩土方量，含累计列，Excel 直接打开
  - `xxx_出错位置.txt` —— 失败桩号清单及原因

## 快速开始

1. 到 [Releases](https://github.com/LL-hua/LL-go/releases) 下载 `hdm.exe` 和 `lot1.xlsx`
2. 放到同一个文件夹
3. 双击 `hdm.exe`
4. 输入桩号（如 `85600`）查询单点，或输入 `all` 批量导出

输出目录为 `<项目名>result/`。

## 输入文件格式（xlsx）

| Sheet 名 | 列含义 |
|---|---|
| 配置 | 键值对：`ClearDepth` / `XyzRows` / `XyzCols` / `投影方式` / `中央子午线` / `南北半球` |
| 平曲线 | 起点桩号, X, Y, 方位角(DDD.MMSS), 长度, 起点半径, 终点半径, 转向 |
| 竖曲线 | 桩号, 高程, 竖曲线半径 |
| 桩号 | 桩号 |
| 原地面 | X, Y, Z |
| 左板块 / 右板块 | 桩号, [宽, 坡]... |
| 边坡 | 起桩号, 止桩号, 类型, 位移串 |
| 左结构层 / 右结构层 | 起桩号, 止桩号, 层号, 名称, 厚度, 内台阶宽, 内坡, 外台阶宽, 外坡 |
| 左结构层横坡 / 右结构层横坡 | 桩号, 横坡(%) |
| 控制点 | 名称, X, Y, Z, 经度, 纬度 |

## 从源码构建

需要 Go 1.21+。

```bash
git clone https://github.com/LL-hua/LL-go.git
cd LL-go

# 本地运行
go run .

# 编译当前平台
go build -o hdm

# 交叉编译 Windows
GOOS=windows GOARCH=amd64 go build -o hdm.exe

# 可选：UPX 压缩
upx -9 hdm.exe
```

## 项目结构

```
.
├── main.go          # 命令行交互 + 批量导出
├── config.go        # AppConfig
├── models.go        # 数据模型
├── ll.go            # 桩号/角度/几何/线路正反算/面积/竖曲线/横坡
├── xlsx.go          # XLSX 读取（archive/zip + encoding/xml 流式解析）
├── parsers.go       # 路基宽度/边坡/横坡/左右结构层 四个引擎
├── section.go       # SectionCalculator 单桩号断面计算核心
├── project.go       # ProjectData + LoadProject
├── html_export.go   # HTML 批量导出（内嵌模板）
└── csv_export.go    # CSV 土方数据导出
```

## 特性

- **纯标准库**：不依赖任何第三方包，`go build` 一遍过
- **单文件分发**：编译后 2~4 MB，用户 PC 无需装 Go / .NET / Java
- **跨平台交叉编译**：`GOOS` + `GOARCH` 一行出一条平台
- **快**：1310 个断面批量计算约 100 ms，单点查询 < 1 ms
- **忠实反映数据问题**：缺数据就报错列清单，不静默兜底

## License

MIT
