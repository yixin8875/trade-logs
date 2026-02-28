package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type xlsxCell struct {
	value    string
	isNumber bool
}

func writeTradeWorkbookXLSX(path string, data tradeDataFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	defer zw.Close()

	tradesRows := make([][]xlsxCell, 0, len(data.Trades))
	for _, row := range data.Trades {
		tradesRows = append(tradesRows, []xlsxCell{
			{value: row.Date},
			{value: row.Note},
			{value: row.EntryReason},
			{value: row.TradeType},
			{value: row.ExitReason},
			{value: row.Supplement},
			{value: fmt.Sprintf("%.4f", row.PositionSize), isNumber: true},
			{value: row.Direction},
			{value: fmt.Sprintf("%.4f", row.EntryPrice), isNumber: true},
			{value: fmt.Sprintf("%.4f", row.ExitPrice1), isNumber: true},
			{value: fmt.Sprintf("%.4f", row.ExitPrice2), isNumber: true},
			{value: fmt.Sprintf("%.4f", row.PnL), isNumber: true},
			{value: row.ErrorReason},
		})
	}

	errorRows := make([][]xlsxCell, 0, len(data.ErrorTypes))
	for _, row := range data.ErrorTypes {
		errorRows = append(errorRows, []xlsxCell{
			{value: row.Reason},
			{value: strconv.Itoa(row.Count), isNumber: true},
			{value: row.ExitReason},
		})
	}

	journalRows := make([][]xlsxCell, 0, len(data.Journals))
	for _, row := range data.Journals {
		journalRows = append(journalRows, []xlsxCell{
			{value: row.Date},
			{value: row.RuleExecuted},
			{value: row.MoodStable},
			{value: row.DidRecord},
			{value: row.Prepared},
			{value: row.NoFOMO},
			{value: fmt.Sprintf("%.4f", row.TotalPnL), isNumber: true},
			{value: row.Note},
		})
	}

	if err := writeZipFile(zw, "[Content_Types].xml", contentTypesXML()); err != nil {
		return err
	}
	if err := writeZipFile(zw, "_rels/.rels", rootRelsXML()); err != nil {
		return err
	}
	if err := writeZipFile(zw, "xl/workbook.xml", workbookXMLString()); err != nil {
		return err
	}
	if err := writeZipFile(zw, "xl/_rels/workbook.xml.rels", workbookRelsXMLString()); err != nil {
		return err
	}
	if err := writeZipFile(zw, "xl/styles.xml", stylesXML()); err != nil {
		return err
	}

	if err := writeZipFile(zw, "xl/worksheets/sheet1.xml", buildWorksheetXML([]string{"日期", "备注", "入场理由", "类型", "离场理由/方式", "补充说明", "仓位大小", "方向", "入场价格", "离场价格1", "离场价格2", "盈亏", "错误原因"}, tradesRows)); err != nil {
		return err
	}
	if err := writeZipFile(zw, "xl/worksheets/sheet2.xml", buildWorksheetXML([]string{"错误原因", "计数", "离场理由"}, errorRows)); err != nil {
		return err
	}
	if err := writeZipFile(zw, "xl/worksheets/sheet3.xml", buildWorksheetXML([]string{"日期", "是否执行规则？", "是否情绪稳定？", "是否做记录？", "是否提前准备？", "没有 FOMO？", "总盈亏", "备注"}, journalRows)); err != nil {
		return err
	}

	return nil
}

func writeZipFile(zw *zip.Writer, name, content string) error {
	writer, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(content))
	return err
}

func contentTypesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/worksheets/sheet3.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
</Types>`
}

func rootRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`
}

func workbookXMLString() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="日志" sheetId="1" r:id="rId1"/>
    <sheet name="错误类型" sheetId="2" r:id="rId2"/>
    <sheet name="别瞎搞日记本" sheetId="3" r:id="rId3"/>
  </sheets>
</workbook>`
}

func workbookRelsXMLString() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet3.xml"/>
  <Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`
}

func stylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts>
  <fills count="1"><fill><patternFill patternType="none"/></fill></fills>
  <borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>
  <cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
  <cellXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/></cellXfs>
</styleSheet>`
}

func buildWorksheetXML(headers []string, rows [][]xlsxCell) string {
	var builder bytes.Buffer
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	builder.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)

	headerCells := make([]xlsxCell, 0, len(headers))
	for _, header := range headers {
		headerCells = append(headerCells, xlsxCell{value: header})
	}
	builder.WriteString(buildRowXML(1, headerCells))

	for idx, row := range rows {
		builder.WriteString(buildRowXML(idx+2, row))
	}

	builder.WriteString(`</sheetData></worksheet>`)
	return builder.String()
}

func buildRowXML(rowNumber int, row []xlsxCell) string {
	var builder bytes.Buffer
	builder.WriteString(`<row r="`)
	builder.WriteString(strconv.Itoa(rowNumber))
	builder.WriteString(`">`)

	for i, cell := range row {
		ref := columnName(i+1) + strconv.Itoa(rowNumber)
		if cell.isNumber && cell.value != "" {
			builder.WriteString(`<c r="`)
			builder.WriteString(ref)
			builder.WriteString(`"><v>`)
			builder.WriteString(escapeXML(cell.value))
			builder.WriteString(`</v></c>`)
			continue
		}
		builder.WriteString(`<c r="`)
		builder.WriteString(ref)
		builder.WriteString(`" t="inlineStr"><is><t xml:space="preserve">`)
		builder.WriteString(escapeXML(cell.value))
		builder.WriteString(`</t></is></c>`)
	}

	builder.WriteString(`</row>`)
	return builder.String()
}

func columnName(index int) string {
	if index <= 0 {
		return "A"
	}
	result := ""
	for index > 0 {
		index--
		result = string(rune('A'+(index%26))) + result
		index /= 26
	}
	return result
}

func escapeXML(input string) string {
	var builder bytes.Buffer
	xml.EscapeText(&builder, []byte(input))
	return builder.String()
}
