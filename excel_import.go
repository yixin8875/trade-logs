package main

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type workbookXML struct {
	Sheets []workbookSheetXML `xml:"sheets>sheet"`
}

type workbookSheetXML struct {
	Name string `xml:"name,attr"`
	RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}

type workbookRelsXML struct {
	Relationships []workbookRelXML `xml:"Relationship"`
}

type workbookRelXML struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
}

type sharedStringsXML struct {
	Items []sharedStringXML `xml:"si"`
}

type sharedStringXML struct {
	Text string               `xml:"t"`
	Runs []sharedStringRunXML `xml:"r"`
}

type sharedStringRunXML struct {
	Text string `xml:"t"`
}

type worksheetXML struct {
	Rows []worksheetRowXML `xml:"sheetData>row"`
}

type worksheetRowXML struct {
	Index int                `xml:"r,attr"`
	Cells []worksheetCellXML `xml:"c"`
}

type worksheetCellXML struct {
	Ref       string             `xml:"r,attr"`
	DataType  string             `xml:"t,attr"`
	Value     string             `xml:"v"`
	InlineStr worksheetInlineXML `xml:"is"`
}

type worksheetInlineXML struct {
	Text string `xml:"t"`
}

func parseTradeWorkbookFromXLSX(path string) (tradeDataFile, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return tradeDataFile{}, err
	}
	defer reader.Close()

	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		files[normalizeZipPath(file.Name)] = file
	}

	var workbook workbookXML
	if err := unmarshalZipXML(files, "xl/workbook.xml", &workbook); err != nil {
		return tradeDataFile{}, err
	}

	var rels workbookRelsXML
	if err := unmarshalZipXML(files, "xl/_rels/workbook.xml.rels", &rels); err != nil {
		return tradeDataFile{}, err
	}

	relMap := make(map[string]string, len(rels.Relationships))
	for _, rel := range rels.Relationships {
		relMap[rel.ID] = resolveWorkbookTarget(rel.Target)
	}

	sheetRows := make(map[string][]worksheetRowXML)
	sharedStrings, _ := loadSharedStrings(files)

	for _, sheet := range workbook.Sheets {
		target := relMap[sheet.RID]
		if target == "" {
			continue
		}
		var worksheet worksheetXML
		if err := unmarshalZipXML(files, target, &worksheet); err != nil {
			continue
		}
		sheetRows[strings.TrimSpace(sheet.Name)] = worksheet.Rows
	}

	seed := time.Now().UnixNano()
	result := tradeDataFile{
		Trades:     parseTradesRows(sheetRows["日志"], sharedStrings, seed),
		ErrorTypes: parseErrorTypeRows(sheetRows["错误类型"], sharedStrings, seed),
		Journals:   parseJournalRows(sheetRows["别瞎搞日记本"], sharedStrings, seed),
	}

	if len(result.Trades) == 0 && len(result.ErrorTypes) == 0 && len(result.Journals) == 0 {
		return tradeDataFile{}, errors.New("未在 Excel 中读取到有效数据")
	}
	return result, nil
}

func parseTradesRows(rows []worksheetRowXML, sharedStrings []string, seed int64) []TradeEntry {
	now := time.Now().Unix()
	entries := make([]TradeEntry, 0, len(rows))

	for idx, row := range rows {
		rowIndex := row.Index
		if rowIndex == 0 {
			rowIndex = idx + 1
		}
		if rowIndex < 3 {
			continue
		}

		cells := toCellMap(row.Cells, sharedStrings)
		entry := TradeEntry{
			ID:           fmt.Sprintf("imp-trade-%d-%d", seed, rowIndex),
			Date:         normalizeDateString(parseExcelDate(cells[1])),
			Note:         strings.TrimSpace(cells[2]),
			EntryReason:  strings.TrimSpace(cells[3]),
			TradeType:    strings.TrimSpace(cells[4]),
			ExitReason:   strings.TrimSpace(cells[5]),
			Supplement:   strings.TrimSpace(cells[6]),
			PositionSize: parseFloatLoose(cells[7]),
			Direction:    strings.TrimSpace(cells[8]),
			EntryPrice:   parseFloatLoose(cells[9]),
			ExitPrice1:   parseFloatLoose(cells[10]),
			ExitPrice2:   parseFloatLoose(cells[11]),
			PnL:          parseFloatLoose(cells[12]),
			ErrorReason:  strings.TrimSpace(cells[16]),
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if !isMeaningfulTrade(entry) {
			continue
		}
		if entry.Date == "" {
			entry.Date = time.Now().Format("2006-01-02")
		}
		if entry.PnL == 0 {
			entry.PnL = calculatePnL(entry)
		}

		entries = append(entries, entry)
	}

	return entries
}

func parseErrorTypeRows(rows []worksheetRowXML, sharedStrings []string, seed int64) []ErrorTypeEntry {
	now := time.Now().Unix()
	entries := make([]ErrorTypeEntry, 0, len(rows))
	for idx, row := range rows {
		rowIndex := row.Index
		if rowIndex == 0 {
			rowIndex = idx + 1
		}
		if rowIndex < 2 {
			continue
		}

		cells := toCellMap(row.Cells, sharedStrings)
		reason := strings.TrimSpace(cells[1])
		exitReason := strings.TrimSpace(cells[4])
		count := parseIntLoose(cells[2])
		if reason == "" && exitReason == "" && count == 0 {
			continue
		}
		entry := ErrorTypeEntry{
			ID:         fmt.Sprintf("imp-err-%d-%d", seed, rowIndex),
			Reason:     reason,
			Count:      count,
			ExitReason: exitReason,
			UpdatedAt:  now,
		}
		entries = append(entries, entry)
	}
	return entries
}

func parseJournalRows(rows []worksheetRowXML, sharedStrings []string, seed int64) []DailyJournalEntry {
	now := time.Now().Unix()
	entries := make([]DailyJournalEntry, 0, len(rows))
	for idx, row := range rows {
		rowIndex := row.Index
		if rowIndex == 0 {
			rowIndex = idx + 1
		}
		if rowIndex < 2 {
			continue
		}

		cells := toCellMap(row.Cells, sharedStrings)
		entry := DailyJournalEntry{
			ID:           fmt.Sprintf("imp-journal-%d-%d", seed, rowIndex),
			Date:         normalizeDateString(parseExcelDate(cells[1])),
			RuleExecuted: strings.TrimSpace(cells[2]),
			MoodStable:   strings.TrimSpace(cells[3]),
			DidRecord:    strings.TrimSpace(cells[4]),
			Prepared:     strings.TrimSpace(cells[5]),
			NoFOMO:       strings.TrimSpace(cells[6]),
			TotalPnL:     parseFloatLoose(cells[7]),
			Note:         strings.TrimSpace(cells[8]),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if !isMeaningfulJournal(entry) {
			continue
		}
		if entry.Date == "" {
			entry.Date = time.Now().Format("2006-01-02")
		}
		entries = append(entries, entry)
	}
	return entries
}

func toCellMap(cells []worksheetCellXML, sharedStrings []string) map[int]string {
	result := make(map[int]string, len(cells))
	for _, cell := range cells {
		col := columnIndexFromRef(cell.Ref)
		if col == 0 {
			continue
		}
		result[col] = decodeCellValue(cell, sharedStrings)
	}
	return result
}

func unmarshalZipXML(files map[string]*zip.File, name string, target any) error {
	file, ok := files[normalizeZipPath(name)]
	if !ok {
		return fmt.Errorf("xlsx entry not found: %s", name)
	}

	handle, err := file.Open()
	if err != nil {
		return err
	}
	defer handle.Close()

	data, err := io.ReadAll(handle)
	if err != nil {
		return err
	}

	return xml.Unmarshal(data, target)
}

func loadSharedStrings(files map[string]*zip.File) ([]string, error) {
	file, ok := files[normalizeZipPath("xl/sharedStrings.xml")]
	if !ok {
		return nil, errors.New("sharedStrings.xml not found")
	}

	handle, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	data, err := io.ReadAll(handle)
	if err != nil {
		return nil, err
	}

	var payload sharedStringsXML
	if err := xml.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	values := make([]string, 0, len(payload.Items))
	for _, item := range payload.Items {
		if strings.TrimSpace(item.Text) != "" {
			values = append(values, item.Text)
			continue
		}
		if len(item.Runs) == 0 {
			values = append(values, "")
			continue
		}

		var textBuilder strings.Builder
		for _, run := range item.Runs {
			textBuilder.WriteString(run.Text)
		}
		values = append(values, textBuilder.String())
	}

	return values, nil
}

func resolveWorkbookTarget(target string) string {
	target = strings.TrimSpace(target)
	if strings.HasPrefix(target, "/") {
		return normalizeZipPath(strings.TrimPrefix(target, "/"))
	}
	return normalizeZipPath(filepath.Join("xl", target))
}

func normalizeZipPath(path string) string {
	return strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "/")
}

func decodeCellValue(cell worksheetCellXML, sharedStrings []string) string {
	switch cell.DataType {
	case "s":
		idx, err := strconv.Atoi(strings.TrimSpace(cell.Value))
		if err != nil || idx < 0 || idx >= len(sharedStrings) {
			return ""
		}
		return strings.TrimSpace(sharedStrings[idx])
	case "inlineStr":
		return strings.TrimSpace(cell.InlineStr.Text)
	case "b":
		if strings.TrimSpace(cell.Value) == "1" {
			return "TRUE"
		}
		return "FALSE"
	default:
		if strings.TrimSpace(cell.Value) != "" {
			return strings.TrimSpace(cell.Value)
		}
		return strings.TrimSpace(cell.InlineStr.Text)
	}
}

func columnIndexFromRef(ref string) int {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0
	}

	col := 0
	for _, char := range ref {
		if char < 'A' || char > 'Z' {
			break
		}
		col = col*26 + int(char-'A'+1)
	}
	return col
}

func parseFloatLoose(value string) float64 {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return 0
	}

	clean = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return -1
		case r == ',' || r == '%' || r == '￥' || r == '$':
			return -1
		default:
			return r
		}
	}, clean)
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return 0
	}

	number, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0
	}
	return number
}

func parseExcelDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if parsed := parseDateValue(raw); parsed.Unix() > 0 {
		return parsed.Format("2006-01-02")
	}

	serial, err := strconv.ParseFloat(raw, 64)
	if err != nil || serial <= 0 {
		return raw
	}

	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	date := base.AddDate(0, 0, int(serial))
	return date.Format("2006-01-02")
}

func isMeaningfulTrade(entry TradeEntry) bool {
	return entry.Note != "" ||
		entry.EntryReason != "" ||
		entry.TradeType != "" ||
		entry.ExitReason != "" ||
		entry.Supplement != "" ||
		entry.PositionSize != 0 ||
		entry.Direction != "" ||
		entry.EntryPrice != 0 ||
		entry.ExitPrice1 != 0 ||
		entry.ExitPrice2 != 0 ||
		entry.PnL != 0 ||
		entry.ErrorReason != ""
}

func isMeaningfulJournal(entry DailyJournalEntry) bool {
	return entry.RuleExecuted != "" || entry.MoodStable != "" || entry.DidRecord != "" ||
		entry.Prepared != "" || entry.NoFOMO != "" || entry.TotalPnL != 0 || entry.Note != ""
}
