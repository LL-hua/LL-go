package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ============================================================
// XlsxReader .xlsx 读取器（archive/zip + encoding/xml 流式解析）
// ============================================================

type XlsxReader struct {
	zip             *zip.ReadCloser
	sheetNameToPath map[string]string
	sharedStrings   []string
}

// NewXlsxReader 打开 xlsx 文件
func NewXlsxReader(path string) (*XlsxReader, error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("打开 xlsx 失败: %w", err)
	}
	r := &XlsxReader{
		zip:             z,
		sheetNameToPath: make(map[string]string),
	}
	if err := r.loadWorkbook(); err != nil {
		z.Close()
		return nil, err
	}
	if err := r.loadSharedStrings(); err != nil {
		// sharedStrings 可能不存在，忽略
	}
	return r, nil
}

// Close 关闭 zip
func (r *XlsxReader) Close() error {
	if r.zip != nil {
		return r.zip.Close()
	}
	return nil
}

// getEntry 找 zip 内文件
func (r *XlsxReader) getEntry(path string) *zip.File {
	for _, f := range r.zip.File {
		if f.Name == path {
			return f
		}
	}
	return nil
}

// readEntry 读整个文件内容
func (r *XlsxReader) readEntry(path string) ([]byte, error) {
	f := r.getEntry(path)
	if f == nil {
		return nil, fmt.Errorf("找不到 zip 内文件: %s", path)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// ============================================================
// 读 workbook.xml + workbook.xml.rels
// ============================================================

const (
	nsMain = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	nsRel  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
)

type xlsxSheet struct {
	Name string `xml:"name,attr"`
	ID   string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}

type xlsxWorkbook struct {
	Sheets struct {
		Sheet []xlsxSheet `xml:"sheet"`
	} `xml:"sheets"`
}

type xlsxRel struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
}

type xlsxRels struct {
	Relationships []xlsxRel `xml:"Relationship"`
}

func (r *XlsxReader) loadWorkbook() error {
	data, err := r.readEntry("xl/workbook.xml")
	if err != nil {
		return err
	}

	var wb xlsxWorkbook
	if err := xml.Unmarshal(data, &wb); err != nil {
		return fmt.Errorf("解析 workbook.xml 失败: %w", err)
	}

	rIdToName := make(map[string]string)
	for _, s := range wb.Sheets.Sheet {
		rIdToName[s.ID] = s.Name
	}

	relData, err := r.readEntry("xl/_rels/workbook.xml.rels")
	if err != nil {
		return err
	}

	var rels xlsxRels
	if err := xml.Unmarshal(relData, &rels); err != nil {
		return fmt.Errorf("解析 workbook.xml.rels 失败: %w", err)
	}

	for _, rel := range rels.Relationships {
		name, ok := rIdToName[rel.ID]
		if !ok {
			continue
		}
		target := strings.TrimPrefix(rel.Target, "/")
		if !strings.HasPrefix(target, "xl/") {
			target = "xl/" + target
		}
		r.sheetNameToPath[name] = target
	}
	return nil
}

// ============================================================
// 读 sharedStrings.xml（流式）
// ============================================================

func (r *XlsxReader) loadSharedStrings() error {
	f := r.getEntry("xl/sharedStrings.xml")
	if f == nil {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	dec := xml.NewDecoder(rc)
	var sb strings.Builder
	inT := false

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				inT = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inT = false
			} else if t.Name.Local == "si" {
				r.sharedStrings = append(r.sharedStrings, sb.String())
				sb.Reset()
			}
		case xml.CharData:
			if inT {
				sb.Write(t)
			}
		}
	}
	return nil
}

// ============================================================
// 读 sheet（流式，逐行解码）
// ============================================================

type xlsxInlineString struct {
	T string `xml:"t"`
}

type xlsxCell struct {
	Ref  string            `xml:"r,attr"`
	Type string            `xml:"t,attr"`
	V    string            `xml:"v"`
	IS   *xlsxInlineString `xml:"is"`
}

type xlsxRow struct {
	R     string     `xml:"r,attr"`
	Cells []xlsxCell `xml:"c"`
}

// ReadSheet 读指定 sheet，返回每行 map[列索引]字符串值（跳过第一行说明行）
func (r *XlsxReader) ReadSheet(sheetName string) ([]map[int]string, error) {
	path, ok := r.sheetNameToPath[sheetName]
	if !ok {
		return nil, fmt.Errorf("找不到 sheet: %s", sheetName)
	}
	f := r.getEntry(path)
	if f == nil {
		return nil, fmt.Errorf("找不到 sheet 文件: %s", path)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	rows := make([]map[int]string, 0)
	dec := xml.NewDecoder(rc)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读 sheet 失败: %s: %w", sheetName, err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "row" {
			continue
		}

		var rd xlsxRow
		if err := dec.DecodeElement(&rd, &se); err != nil {
			return nil, err
		}

		// 跳过第 1 行（说明行）
		if rd.R == "1" {
			continue
		}

		m := make(map[int]string)
		for _, c := range rd.Cells {
			colIdx := ColRefToIndex(c.Ref)
			var val string
			switch c.Type {
			case "s":
				idx, err := strconv.Atoi(strings.TrimSpace(c.V))
				if err == nil && idx >= 0 && idx < len(r.sharedStrings) {
					val = r.sharedStrings[idx]
				}
			case "inlineStr":
				if c.IS != nil {
					val = c.IS.T
				}
			default:
				val = c.V
			}
			if val != "" {
				m[colIdx] = val
			}
		}
		rows = append(rows, m)
	}
	return rows, nil
}

// ============================================================
// 工具
// ============================================================

// ColRefToIndex "A"->0, "Z"->25, "AA"->26, ... 从 "A1" 里提取字母部分
func ColRefToIndex(ref string) int {
	if ref == "" {
		return 0
	}
	col := 0
	for i := 0; i < len(ref); i++ {
		ch := ref[i]
		if ch >= 'A' && ch <= 'Z' {
			col = col*26 + int(ch-'A'+1)
		} else {
			break
		}
	}
	return col - 1
}
