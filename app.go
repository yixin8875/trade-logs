package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const defaultImportPath = "/Users/boohee/Downloads/交易日志模板菜真寒版.xlsx"

// App is the root bound object exposed to frontend.
type App struct {
	ctx     context.Context
	store   *tradeStore
	initErr error
}

type TradeEntry struct {
	ID           string  `json:"id"`
	Date         string  `json:"date"`
	Note         string  `json:"note"`
	EntryReason  string  `json:"entryReason"`
	TradeType    string  `json:"tradeType"`
	ExitReason   string  `json:"exitReason"`
	Supplement   string  `json:"supplement"`
	PositionSize float64 `json:"positionSize"`
	Direction    string  `json:"direction"`
	EntryPrice   float64 `json:"entryPrice"`
	ExitPrice1   float64 `json:"exitPrice1"`
	ExitPrice2   float64 `json:"exitPrice2"`
	PnL          float64 `json:"pnl"`
	ErrorReason  string  `json:"errorReason"`
	CreatedAt    int64   `json:"createdAt"`
	UpdatedAt    int64   `json:"updatedAt"`
}

type ErrorTypeEntry struct {
	ID         string `json:"id"`
	Reason     string `json:"reason"`
	Count      int    `json:"count"`
	ExitReason string `json:"exitReason"`
	UpdatedAt  int64  `json:"updatedAt"`
}

type DailyJournalEntry struct {
	ID           string  `json:"id"`
	Date         string  `json:"date"`
	RuleExecuted string  `json:"ruleExecuted"`
	MoodStable   string  `json:"moodStable"`
	DidRecord    string  `json:"didRecord"`
	Prepared     string  `json:"prepared"`
	NoFOMO       string  `json:"noFOMO"`
	TotalPnL     float64 `json:"totalPnL"`
	Note         string  `json:"note"`
	CreatedAt    int64   `json:"createdAt"`
	UpdatedAt    int64   `json:"updatedAt"`
}

type TradeSummary struct {
	TotalTrades      int     `json:"totalTrades"`
	Wins             int     `json:"wins"`
	Losses           int     `json:"losses"`
	Breakeven        int     `json:"breakeven"`
	WinRate          float64 `json:"winRate"`
	TotalPnL         float64 `json:"totalPnL"`
	AvgWin           float64 `json:"avgWin"`
	AvgLoss          float64 `json:"avgLoss"`
	ProfitLossRatio  float64 `json:"profitLossRatio"`
	AveragePosition  float64 `json:"averagePosition"`
	AverageHoldRange float64 `json:"averageHoldRange"`
}

type TradeDashboard struct {
	Trades            []TradeEntry        `json:"trades"`
	ErrorTypes        []ErrorTypeEntry    `json:"errorTypes"`
	Journals          []DailyJournalEntry `json:"journals"`
	Summary           TradeSummary        `json:"summary"`
	DataFile          string              `json:"dataFile"`
	DefaultImportPath string              `json:"defaultImportPath"`
	DefaultExportDir  string              `json:"defaultExportDir"`
}

type tradeDataFile struct {
	Trades     []TradeEntry        `json:"trades"`
	ErrorTypes []ErrorTypeEntry    `json:"errorTypes"`
	Journals   []DailyJournalEntry `json:"journals"`
}

type tradeStore struct {
	mu     sync.Mutex
	db     *sql.DB
	dbPath string
}

func NewApp() *App {
	return &App{}
}

func newTradeStore(path string) (*tradeStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	store := &tradeStore{db: db, dbPath: path}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *tradeStore) initSchema() error {
	statements := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		`CREATE TABLE IF NOT EXISTS trades (
			id TEXT PRIMARY KEY,
			date TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			entry_reason TEXT NOT NULL DEFAULT '',
			trade_type TEXT NOT NULL DEFAULT '',
			exit_reason TEXT NOT NULL DEFAULT '',
			supplement TEXT NOT NULL DEFAULT '',
			position_size REAL NOT NULL DEFAULT 0,
			direction TEXT NOT NULL DEFAULT '',
			entry_price REAL NOT NULL DEFAULT 0,
			exit_price1 REAL NOT NULL DEFAULT 0,
			exit_price2 REAL NOT NULL DEFAULT 0,
			pnl REAL NOT NULL DEFAULT 0,
			error_reason TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_trades_date_updated ON trades(date DESC, updated_at DESC);`,
		`CREATE TABLE IF NOT EXISTS error_types (
			id TEXT PRIMARY KEY,
			reason TEXT NOT NULL DEFAULT '',
			count INTEGER NOT NULL DEFAULT 0,
			exit_reason TEXT NOT NULL DEFAULT '',
			updated_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS journals (
			id TEXT PRIMARY KEY,
			date TEXT NOT NULL,
			rule_executed TEXT NOT NULL DEFAULT '',
			mood_stable TEXT NOT NULL DEFAULT '',
			did_record TEXT NOT NULL DEFAULT '',
			prepared TEXT NOT NULL DEFAULT '',
			no_fomo TEXT NOT NULL DEFAULT '',
			total_pnl REAL NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_journals_date_updated ON journals(date DESC, updated_at DESC);`,
	}

	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *tradeStore) countAllRows() (int, error) {
	queries := []string{
		"SELECT COUNT(1) FROM trades;",
		"SELECT COUNT(1) FROM error_types;",
		"SELECT COUNT(1) FROM journals;",
	}
	count := 0
	for _, query := range queries {
		var c int
		if err := s.db.QueryRow(query).Scan(&c); err != nil {
			return 0, err
		}
		count += c
	}
	return count, nil
}

func (s *tradeStore) migrateFromLegacyJSON(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	total, err := s.countAllRows()
	if err != nil {
		return err
	}
	if total > 0 {
		return nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	payload := tradeDataFile{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		var tradesOnly []TradeEntry
		if err := json.Unmarshal(raw, &tradesOnly); err != nil {
			return nil
		}
		payload.Trades = tradesOnly
	}

	for i := range payload.Trades {
		if payload.Trades[i].ID == "" {
			payload.Trades[i].ID = generateRecordID("trade")
		}
		if payload.Trades[i].CreatedAt == 0 {
			payload.Trades[i].CreatedAt = time.Now().Unix()
		}
		if payload.Trades[i].UpdatedAt == 0 {
			payload.Trades[i].UpdatedAt = payload.Trades[i].CreatedAt
		}
	}
	for i := range payload.ErrorTypes {
		if payload.ErrorTypes[i].ID == "" {
			payload.ErrorTypes[i].ID = generateRecordID("err")
		}
		if payload.ErrorTypes[i].UpdatedAt == 0 {
			payload.ErrorTypes[i].UpdatedAt = time.Now().Unix()
		}
	}
	for i := range payload.Journals {
		if payload.Journals[i].ID == "" {
			payload.Journals[i].ID = generateRecordID("journal")
		}
		if payload.Journals[i].CreatedAt == 0 {
			payload.Journals[i].CreatedAt = time.Now().Unix()
		}
		if payload.Journals[i].UpdatedAt == 0 {
			payload.Journals[i].UpdatedAt = payload.Journals[i].CreatedAt
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := replaceAllDataTx(tx, payload); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *tradeStore) readAllData() (tradeDataFile, error) {
	result := tradeDataFile{
		Trades:     make([]TradeEntry, 0),
		ErrorTypes: make([]ErrorTypeEntry, 0),
		Journals:   make([]DailyJournalEntry, 0),
	}

	tradesRows, err := s.db.Query(`SELECT id, date, note, entry_reason, trade_type, exit_reason, supplement, position_size, direction, entry_price, exit_price1, exit_price2, pnl, error_reason, created_at, updated_at FROM trades ORDER BY date DESC, updated_at DESC`)
	if err != nil {
		return result, err
	}
	for tradesRows.Next() {
		var item TradeEntry
		if err := tradesRows.Scan(
			&item.ID,
			&item.Date,
			&item.Note,
			&item.EntryReason,
			&item.TradeType,
			&item.ExitReason,
			&item.Supplement,
			&item.PositionSize,
			&item.Direction,
			&item.EntryPrice,
			&item.ExitPrice1,
			&item.ExitPrice2,
			&item.PnL,
			&item.ErrorReason,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			tradesRows.Close()
			return result, err
		}
		result.Trades = append(result.Trades, item)
	}
	if err := tradesRows.Err(); err != nil {
		tradesRows.Close()
		return result, err
	}
	tradesRows.Close()

	errorRows, err := s.db.Query(`SELECT id, reason, count, exit_reason, updated_at FROM error_types ORDER BY count DESC, updated_at DESC`)
	if err != nil {
		return result, err
	}
	for errorRows.Next() {
		var item ErrorTypeEntry
		if err := errorRows.Scan(&item.ID, &item.Reason, &item.Count, &item.ExitReason, &item.UpdatedAt); err != nil {
			errorRows.Close()
			return result, err
		}
		result.ErrorTypes = append(result.ErrorTypes, item)
	}
	if err := errorRows.Err(); err != nil {
		errorRows.Close()
		return result, err
	}
	errorRows.Close()

	journalRows, err := s.db.Query(`SELECT id, date, rule_executed, mood_stable, did_record, prepared, no_fomo, total_pnl, note, created_at, updated_at FROM journals ORDER BY date DESC, updated_at DESC`)
	if err != nil {
		return result, err
	}
	for journalRows.Next() {
		var item DailyJournalEntry
		if err := journalRows.Scan(
			&item.ID,
			&item.Date,
			&item.RuleExecuted,
			&item.MoodStable,
			&item.DidRecord,
			&item.Prepared,
			&item.NoFOMO,
			&item.TotalPnL,
			&item.Note,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			journalRows.Close()
			return result, err
		}
		result.Journals = append(result.Journals, item)
	}
	if err := journalRows.Err(); err != nil {
		journalRows.Close()
		return result, err
	}
	journalRows.Close()

	return result, nil
}

func (s *tradeStore) dashboard() (TradeDashboard, error) {
	snapshot, err := s.readAllData()
	if err != nil {
		return TradeDashboard{}, err
	}
	return TradeDashboard{
		Trades:            snapshot.Trades,
		ErrorTypes:        snapshot.ErrorTypes,
		Journals:          snapshot.Journals,
		Summary:           buildSummary(snapshot.Trades),
		DataFile:          s.dbPath,
		DefaultImportPath: defaultImportPath,
		DefaultExportDir:  defaultExportDir(),
	}, nil
}

func dbFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		return filepath.Join(".", "data", "trades.db")
	}
	return filepath.Join(configDir, "trade-logs", "trades.db")
}

func legacyJSONPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		return filepath.Join(".", "data", "trades.json")
	}
	return filepath.Join(configDir, "trade-logs", "trades.json")
}

func defaultExportDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	return filepath.Join(home, "Downloads")
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	store, err := newTradeStore(dbFilePath())
	if err != nil {
		a.initErr = err
		return
	}
	a.store = store
	_ = store.migrateFromLegacyJSON(legacyJSONPath())
}

func (a *App) ensureStore() error {
	if a.initErr != nil {
		return a.initErr
	}
	if a.store == nil {
		return errors.New("store not initialized")
	}
	return nil
}

func (a *App) GetDashboard() (TradeDashboard, error) {
	if err := a.ensureStore(); err != nil {
		return TradeDashboard{}, err
	}
	return a.store.dashboard()
}

func (a *App) SaveTrade(input TradeEntry) (TradeDashboard, error) {
	if err := a.ensureStore(); err != nil {
		return TradeDashboard{}, err
	}

	now := time.Now().Unix()
	cleaned := sanitizeTrade(input)
	if cleaned.ID == "" {
		cleaned.ID = generateRecordID("trade")
		cleaned.CreatedAt = now
	}
	cleaned.UpdatedAt = now
	if cleaned.CreatedAt == 0 {
		cleaned.CreatedAt = now
	}
	if cleaned.Date == "" {
		cleaned.Date = time.Now().Format("2006-01-02")
	}
	if cleaned.PnL == 0 {
		cleaned.PnL = calculatePnL(cleaned)
	}

	a.store.mu.Lock()
	defer a.store.mu.Unlock()

	_, err := a.store.db.Exec(`
	INSERT INTO trades (id, date, note, entry_reason, trade_type, exit_reason, supplement, position_size, direction, entry_price, exit_price1, exit_price2, pnl, error_reason, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		date=excluded.date,
		note=excluded.note,
		entry_reason=excluded.entry_reason,
		trade_type=excluded.trade_type,
		exit_reason=excluded.exit_reason,
		supplement=excluded.supplement,
		position_size=excluded.position_size,
		direction=excluded.direction,
		entry_price=excluded.entry_price,
		exit_price1=excluded.exit_price1,
		exit_price2=excluded.exit_price2,
		pnl=excluded.pnl,
		error_reason=excluded.error_reason,
		created_at=excluded.created_at,
		updated_at=excluded.updated_at;
	`,
		cleaned.ID,
		cleaned.Date,
		cleaned.Note,
		cleaned.EntryReason,
		cleaned.TradeType,
		cleaned.ExitReason,
		cleaned.Supplement,
		cleaned.PositionSize,
		cleaned.Direction,
		cleaned.EntryPrice,
		cleaned.ExitPrice1,
		cleaned.ExitPrice2,
		cleaned.PnL,
		cleaned.ErrorReason,
		cleaned.CreatedAt,
		cleaned.UpdatedAt,
	)
	if err != nil {
		return TradeDashboard{}, err
	}

	return a.store.dashboard()
}

func (a *App) DeleteTrade(id string) (TradeDashboard, error) {
	if err := a.ensureStore(); err != nil {
		return TradeDashboard{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return TradeDashboard{}, errors.New("id is required")
	}

	a.store.mu.Lock()
	defer a.store.mu.Unlock()

	if _, err := a.store.db.Exec(`DELETE FROM trades WHERE id = ?`, id); err != nil {
		return TradeDashboard{}, err
	}
	return a.store.dashboard()
}

func (a *App) SaveErrorType(input ErrorTypeEntry) (TradeDashboard, error) {
	if err := a.ensureStore(); err != nil {
		return TradeDashboard{}, err
	}

	now := time.Now().Unix()
	input.ID = strings.TrimSpace(input.ID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.ExitReason = strings.TrimSpace(input.ExitReason)
	input.UpdatedAt = now
	if input.Count < 0 {
		input.Count = 0
	}
	if input.ID == "" {
		input.ID = generateRecordID("err")
	}

	a.store.mu.Lock()
	defer a.store.mu.Unlock()

	_, err := a.store.db.Exec(`
	INSERT INTO error_types (id, reason, count, exit_reason, updated_at)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		reason=excluded.reason,
		count=excluded.count,
		exit_reason=excluded.exit_reason,
		updated_at=excluded.updated_at;
	`, input.ID, input.Reason, input.Count, input.ExitReason, input.UpdatedAt)
	if err != nil {
		return TradeDashboard{}, err
	}
	return a.store.dashboard()
}

func (a *App) DeleteErrorType(id string) (TradeDashboard, error) {
	if err := a.ensureStore(); err != nil {
		return TradeDashboard{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return TradeDashboard{}, errors.New("id is required")
	}

	a.store.mu.Lock()
	defer a.store.mu.Unlock()

	if _, err := a.store.db.Exec(`DELETE FROM error_types WHERE id = ?`, id); err != nil {
		return TradeDashboard{}, err
	}
	return a.store.dashboard()
}

func (a *App) SaveJournal(input DailyJournalEntry) (TradeDashboard, error) {
	if err := a.ensureStore(); err != nil {
		return TradeDashboard{}, err
	}

	now := time.Now().Unix()
	input.ID = strings.TrimSpace(input.ID)
	input.Date = normalizeDateString(input.Date)
	input.RuleExecuted = strings.TrimSpace(input.RuleExecuted)
	input.MoodStable = strings.TrimSpace(input.MoodStable)
	input.DidRecord = strings.TrimSpace(input.DidRecord)
	input.Prepared = strings.TrimSpace(input.Prepared)
	input.NoFOMO = strings.TrimSpace(input.NoFOMO)
	input.Note = strings.TrimSpace(input.Note)
	if input.ID == "" {
		input.ID = generateRecordID("journal")
		input.CreatedAt = now
	}
	if input.CreatedAt == 0 {
		input.CreatedAt = now
	}
	input.UpdatedAt = now
	if input.Date == "" {
		input.Date = time.Now().Format("2006-01-02")
	}

	a.store.mu.Lock()
	defer a.store.mu.Unlock()

	_, err := a.store.db.Exec(`
	INSERT INTO journals (id, date, rule_executed, mood_stable, did_record, prepared, no_fomo, total_pnl, note, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		date=excluded.date,
		rule_executed=excluded.rule_executed,
		mood_stable=excluded.mood_stable,
		did_record=excluded.did_record,
		prepared=excluded.prepared,
		no_fomo=excluded.no_fomo,
		total_pnl=excluded.total_pnl,
		note=excluded.note,
		created_at=excluded.created_at,
		updated_at=excluded.updated_at;
	`, input.ID, input.Date, input.RuleExecuted, input.MoodStable, input.DidRecord, input.Prepared, input.NoFOMO, input.TotalPnL, input.Note, input.CreatedAt, input.UpdatedAt)
	if err != nil {
		return TradeDashboard{}, err
	}
	return a.store.dashboard()
}

func (a *App) DeleteJournal(id string) (TradeDashboard, error) {
	if err := a.ensureStore(); err != nil {
		return TradeDashboard{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return TradeDashboard{}, errors.New("id is required")
	}

	a.store.mu.Lock()
	defer a.store.mu.Unlock()

	if _, err := a.store.db.Exec(`DELETE FROM journals WHERE id = ?`, id); err != nil {
		return TradeDashboard{}, err
	}
	return a.store.dashboard()
}

func replaceAllDataTx(tx *sql.Tx, payload tradeDataFile) error {
	if _, err := tx.Exec(`DELETE FROM trades`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM error_types`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM journals`); err != nil {
		return err
	}

	for _, item := range payload.Trades {
		if _, err := tx.Exec(`
			INSERT INTO trades (id, date, note, entry_reason, trade_type, exit_reason, supplement, position_size, direction, entry_price, exit_price1, exit_price2, pnl, error_reason, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.ID,
			item.Date,
			item.Note,
			item.EntryReason,
			item.TradeType,
			item.ExitReason,
			item.Supplement,
			item.PositionSize,
			item.Direction,
			item.EntryPrice,
			item.ExitPrice1,
			item.ExitPrice2,
			item.PnL,
			item.ErrorReason,
			item.CreatedAt,
			item.UpdatedAt,
		); err != nil {
			return err
		}
	}

	for _, item := range payload.ErrorTypes {
		if _, err := tx.Exec(`
			INSERT INTO error_types (id, reason, count, exit_reason, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, item.ID, item.Reason, item.Count, item.ExitReason, item.UpdatedAt); err != nil {
			return err
		}
	}

	for _, item := range payload.Journals {
		if _, err := tx.Exec(`
			INSERT INTO journals (id, date, rule_executed, mood_stable, did_record, prepared, no_fomo, total_pnl, note, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, item.ID, item.Date, item.RuleExecuted, item.MoodStable, item.DidRecord, item.Prepared, item.NoFOMO, item.TotalPnL, item.Note, item.CreatedAt, item.UpdatedAt); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) ImportTradesFromExcel(path string) (TradeDashboard, error) {
	if err := a.ensureStore(); err != nil {
		return TradeDashboard{}, err
	}

	path = strings.TrimSpace(path)
	if path == "" {
		path = defaultImportPath
	}

	payload, err := parseTradeWorkbookFromXLSX(path)
	if err != nil {
		return TradeDashboard{}, err
	}

	now := time.Now().Unix()
	for i := range payload.Trades {
		if payload.Trades[i].ID == "" {
			payload.Trades[i].ID = generateRecordID("trade")
		}
		if payload.Trades[i].CreatedAt == 0 {
			payload.Trades[i].CreatedAt = now
		}
		if payload.Trades[i].UpdatedAt == 0 {
			payload.Trades[i].UpdatedAt = now
		}
		if payload.Trades[i].PnL == 0 {
			payload.Trades[i].PnL = calculatePnL(payload.Trades[i])
		}
	}
	for i := range payload.ErrorTypes {
		if payload.ErrorTypes[i].ID == "" {
			payload.ErrorTypes[i].ID = generateRecordID("err")
		}
		if payload.ErrorTypes[i].UpdatedAt == 0 {
			payload.ErrorTypes[i].UpdatedAt = now
		}
	}
	for i := range payload.Journals {
		if payload.Journals[i].ID == "" {
			payload.Journals[i].ID = generateRecordID("journal")
		}
		if payload.Journals[i].CreatedAt == 0 {
			payload.Journals[i].CreatedAt = now
		}
		if payload.Journals[i].UpdatedAt == 0 {
			payload.Journals[i].UpdatedAt = now
		}
	}

	a.store.mu.Lock()
	defer a.store.mu.Unlock()

	tx, err := a.store.db.Begin()
	if err != nil {
		return TradeDashboard{}, err
	}
	defer tx.Rollback()

	if err := replaceAllDataTx(tx, payload); err != nil {
		return TradeDashboard{}, err
	}
	if err := tx.Commit(); err != nil {
		return TradeDashboard{}, err
	}

	return a.store.dashboard()
}

func (a *App) ExportTradesToCSV(path string) (string, error) {
	if err := a.ensureStore(); err != nil {
		return "", err
	}

	snapshot, err := a.store.readAllData()
	if err != nil {
		return "", err
	}

	target := strings.TrimSpace(path)
	if target == "" {
		target = filepath.Join(defaultExportDir(), fmt.Sprintf("trade-logs-%s.csv", time.Now().Format("20060102-150405")))
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}

	file, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"日期", "备注", "入场理由", "类型", "离场理由/方式", "补充说明", "仓位大小", "方向", "入场价格", "离场价格1", "离场价格2", "盈亏", "错误原因"}
	if err := writer.Write(header); err != nil {
		return "", err
	}

	for _, trade := range snapshot.Trades {
		row := []string{
			trade.Date,
			trade.Note,
			trade.EntryReason,
			trade.TradeType,
			trade.ExitReason,
			trade.Supplement,
			fmt.Sprintf("%.4f", trade.PositionSize),
			trade.Direction,
			fmt.Sprintf("%.4f", trade.EntryPrice),
			fmt.Sprintf("%.4f", trade.ExitPrice1),
			fmt.Sprintf("%.4f", trade.ExitPrice2),
			fmt.Sprintf("%.4f", trade.PnL),
			trade.ErrorReason,
		}
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}

	if err := writer.Error(); err != nil {
		return "", err
	}

	return target, nil
}

func (a *App) ExportDataToExcel(path string) (string, error) {
	if err := a.ensureStore(); err != nil {
		return "", err
	}

	snapshot, err := a.store.readAllData()
	if err != nil {
		return "", err
	}

	target := strings.TrimSpace(path)
	if target == "" {
		target = filepath.Join(defaultExportDir(), fmt.Sprintf("trade-logs-%s.xlsx", time.Now().Format("20060102-150405")))
	}
	if !strings.HasSuffix(strings.ToLower(target), ".xlsx") {
		target += ".xlsx"
	}

	if err := writeTradeWorkbookXLSX(target, snapshot); err != nil {
		return "", err
	}
	return target, nil
}

func sanitizeTrade(input TradeEntry) TradeEntry {
	input.ID = strings.TrimSpace(input.ID)
	input.Date = normalizeDateString(input.Date)
	input.Note = strings.TrimSpace(input.Note)
	input.EntryReason = strings.TrimSpace(input.EntryReason)
	input.TradeType = strings.TrimSpace(input.TradeType)
	input.ExitReason = strings.TrimSpace(input.ExitReason)
	input.Supplement = strings.TrimSpace(input.Supplement)
	input.Direction = strings.TrimSpace(input.Direction)
	input.ErrorReason = strings.TrimSpace(input.ErrorReason)
	return input
}

func buildSummary(trades []TradeEntry) TradeSummary {
	summary := TradeSummary{TotalTrades: len(trades)}
	if len(trades) == 0 {
		return summary
	}

	var winSum float64
	var lossSum float64
	var posSum float64
	var rangeSum float64

	for _, trade := range trades {
		pnl := trade.PnL
		switch {
		case pnl > 0:
			summary.Wins++
			winSum += pnl
		case pnl < 0:
			summary.Losses++
			lossSum += pnl
		default:
			summary.Breakeven++
		}
		summary.TotalPnL += pnl
		posSum += trade.PositionSize
		if trade.EntryPrice > 0 && trade.ExitPrice1 > 0 {
			rangeSum += math.Abs(trade.ExitPrice1 - trade.EntryPrice)
		}
	}

	if summary.TotalTrades > 0 {
		summary.WinRate = float64(summary.Wins) / float64(summary.TotalTrades) * 100
		summary.AveragePosition = posSum / float64(summary.TotalTrades)
		summary.AverageHoldRange = rangeSum / float64(summary.TotalTrades)
	}
	if summary.Wins > 0 {
		summary.AvgWin = winSum / float64(summary.Wins)
	}
	if summary.Losses > 0 {
		summary.AvgLoss = lossSum / float64(summary.Losses)
	}
	if summary.AvgLoss != 0 {
		summary.ProfitLossRatio = summary.AvgWin / math.Abs(summary.AvgLoss)
	}

	return summary
}

func generateRecordID(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "id"
	}
	return prefix + "-" + strings.ReplaceAll(time.Now().UTC().Format("20060102T150405.000000000"), ".", "")
}

func calculatePnL(trade TradeEntry) float64 {
	if trade.PositionSize == 0 || trade.EntryPrice == 0 {
		return trade.PnL
	}

	exit := trade.ExitPrice1
	if exit == 0 && trade.ExitPrice2 > 0 {
		exit = trade.ExitPrice2
	}
	if exit == 0 {
		return trade.PnL
	}

	direction := strings.ToLower(strings.TrimSpace(trade.Direction))
	switch direction {
	case "空", "short", "sell", "做空":
		return (trade.EntryPrice - exit) * trade.PositionSize
	default:
		return (exit - trade.EntryPrice) * trade.PositionSize
	}
}

func parseDateValue(value string) time.Time {
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"2006/1/2",
		"2006-1-2",
		time.RFC3339,
	}
	trimmed := strings.TrimSpace(value)
	for _, format := range formats {
		if t, err := time.Parse(format, trimmed); err == nil {
			return t
		}
	}
	return time.Unix(0, 0)
}

func normalizeDateString(value string) string {
	date := parseDateValue(value)
	if date.Unix() <= 0 {
		return strings.TrimSpace(value)
	}
	return date.Format("2006-01-02")
}

func parseIntLoose(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0
	}
	return int(math.Round(f))
}
