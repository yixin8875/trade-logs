import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { main } from "../wailsjs/go/models";
import {
  DeleteErrorType,
  DeleteJournal,
  DeleteTrade,
  ExportDataToExcel,
  ExportTradesToCSV,
  GetDashboard,
  ImportTradesFromExcel,
  SaveErrorType,
  SaveJournal,
  SaveTrade,
} from "../wailsjs/go/main/App";

type DirectionFilter = "all" | "多" | "空";
type ActiveTab = "trades" | "journals" | "settings";
type SelectOption = { value: string; label: string };
type ConfirmTarget =
  | { kind: "trade"; id: string }
  | { kind: "journal"; id: string }
  | { kind: "errorOption"; item: main.ErrorTypeEntry }
  | { kind: "exitOption"; item: main.ErrorTypeEntry };

type ConfirmState = {
  open: boolean;
  title: string;
  detail: string;
  target: ConfirmTarget | null;
};

type TradeForm = {
  id: string;
  date: string;
  note: string;
  entryReason: string;
  tradeType: string;
  exitReason: string;
  supplement: string;
  positionSize: string;
  direction: string;
  entryPrice: string;
  exitPrice1: string;
  exitPrice2: string;
  pnl: string;
  errorReason: string;
  createdAt: number;
};

type JournalForm = {
  id: string;
  date: string;
  ruleExecuted: string;
  moodStable: string;
  didRecord: string;
  prepared: string;
  noFOMO: string;
  totalPnL: string;
  note: string;
  createdAt: number;
};

const todayString = (): string => new Date().toISOString().slice(0, 10);

const emptyTradeForm = (): TradeForm => ({
  id: "",
  date: todayString(),
  note: "",
  entryReason: "",
  tradeType: "",
  exitReason: "",
  supplement: "",
  positionSize: "1",
  direction: "多",
  entryPrice: "",
  exitPrice1: "",
  exitPrice2: "",
  pnl: "",
  errorReason: "",
  createdAt: 0,
});

const emptyJournalForm = (): JournalForm => ({
  id: "",
  date: todayString(),
  ruleExecuted: "✅",
  moodStable: "✅",
  didRecord: "✅",
  prepared: "✅",
  noFOMO: "✅",
  totalPnL: "0",
  note: "",
  createdAt: 0,
});

const parseNumber = (value: string): number => {
  const parsed = Number.parseFloat(value);
  return Number.isFinite(parsed) ? parsed : 0;
};

const formatAmount = (value: number, digits = 2): string =>
  Number.isFinite(value) ? value.toFixed(digits) : "0.00";

const directionOptions: SelectOption[] = [
  { value: "多", label: "多" },
  { value: "空", label: "空" },
];

const directionFilterOptions: SelectOption[] = [
  { value: "all", label: "全部方向" },
  { value: "多", label: "只看多单" },
  { value: "空", label: "只看空单" },
];

const yesNoOptions = (yesLabel: string, noLabel: string): SelectOption[] => [
  { value: "✅", label: yesLabel },
  { value: "❌", label: noLabel },
];

function CustomSelect(props: {
  value: string;
  options: SelectOption[];
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  searchable?: boolean;
  searchPlaceholder?: string;
}) {
  const {
    value,
    options,
    onChange,
    placeholder = "请选择",
    className = "",
    searchable = false,
    searchPlaceholder = "搜索选项",
  } = props;
  const [open, setOpen] = useState(false);
  const [keyword, setKeyword] = useState("");
  const rootRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const onMouseDown = (event: MouseEvent) => {
      if (!rootRef.current) {
        return;
      }
      if (!rootRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    window.addEventListener("mousedown", onMouseDown);
    return () => window.removeEventListener("mousedown", onMouseDown);
  }, []);

  useEffect(() => {
    if (!open) {
      setKeyword("");
    }
  }, [open]);

  const current = options.find((item) => item.value === value);
  const filteredOptions = useMemo(() => {
    const key = keyword.trim().toLowerCase();
    if (!searchable || key === "") {
      return options;
    }
    return options.filter((item) => {
      const text = `${item.label} ${item.value}`.toLowerCase();
      return text.includes(key);
    });
  }, [keyword, options, searchable]);

  return (
    <div className={`relative ${className}`} ref={rootRef}>
      <button
        type="button"
        className={`input-base flex w-full items-center justify-between gap-2 pr-2 text-left ${
          current ? "text-slate-800" : "text-slate-400"
        }`}
        onClick={() => setOpen((prev) => !prev)}
      >
        <span className="truncate">{current?.label || placeholder}</span>
        <span className={`text-xs text-slate-500 transition ${open ? "rotate-180" : ""}`}>▾</span>
      </button>

      {open && (
        <div className="absolute z-30 mt-1 max-h-64 w-full overflow-auto rounded-sm border border-slate-200 bg-white py-1 shadow-lg">
          {searchable && (
            <div className="sticky top-0 z-10 border-b border-slate-100 bg-white p-2">
              <input
                className="input-base h-9 w-full text-sm"
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Escape") {
                    setOpen(false);
                  }
                }}
                placeholder={searchPlaceholder}
              />
            </div>
          )}

          {filteredOptions.map((item) => {
            const selected = item.value === value;
            return (
              <button
                key={`${item.value}-${item.label}`}
                type="button"
                className={`flex w-full items-center justify-between px-3 py-2 text-left text-sm ${
                  selected ? "bg-slate-100 text-slate-900" : "text-slate-700 hover:bg-slate-50"
                }`}
                onClick={() => {
                  onChange(item.value);
                  setOpen(false);
                }}
              >
                <span className="truncate">{item.label}</span>
                {selected && <span className="text-xs text-slate-500">✓</span>}
              </button>
            );
          })}

          {filteredOptions.length === 0 && (
            <div className="px-3 py-6 text-center text-xs text-slate-500">无匹配项</div>
          )}
        </div>
      )}
    </div>
  );
}

function App() {
  const [dashboard, setDashboard] = useState<main.TradeDashboard | null>(null);
  const [tradeForm, setTradeForm] = useState<TradeForm>(emptyTradeForm);
  const [journalForm, setJournalForm] = useState<JournalForm>(emptyJournalForm);
  const [searchKeyword, setSearchKeyword] = useState("");
  const [directionFilter, setDirectionFilter] = useState<DirectionFilter>("all");
  const [importPath, setImportPath] = useState("");
  const [exportPath, setExportPath] = useState("");
  const [newErrorOption, setNewErrorOption] = useState("");
  const [newExitOption, setNewExitOption] = useState("");
  const [message, setMessage] = useState("正在读取交易日志...");
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<ActiveTab>("trades");
  const [confirmState, setConfirmState] = useState<ConfirmState>({
    open: false,
    title: "",
    detail: "",
    target: null,
  });

  const summary = dashboard?.summary ?? new main.TradeSummary();
  const trades = dashboard?.trades ?? [];
  const errorTypes = dashboard?.errorTypes ?? [];
  const journals = dashboard?.journals ?? [];

  const errorOptionRows = useMemo(() => {
    const map = new Map<string, main.ErrorTypeEntry>();
    errorTypes.forEach((item) => {
      const key = (item.reason || "").trim();
      if (!key) {
        return;
      }
      const exists = map.get(key);
      if (!exists || (item.updatedAt || 0) > (exists.updatedAt || 0)) {
        map.set(key, item);
      }
    });
    return Array.from(map.values()).sort((a, b) =>
      (a.reason || "").localeCompare(b.reason || "", "zh-CN")
    );
  }, [errorTypes]);

  const exitOptionRows = useMemo(() => {
    const map = new Map<string, main.ErrorTypeEntry>();
    errorTypes.forEach((item) => {
      const key = (item.exitReason || "").trim();
      if (!key) {
        return;
      }
      const exists = map.get(key);
      if (!exists || (item.updatedAt || 0) > (exists.updatedAt || 0)) {
        map.set(key, item);
      }
    });
    return Array.from(map.values()).sort((a, b) =>
      (a.exitReason || "").localeCompare(b.exitReason || "", "zh-CN")
    );
  }, [errorTypes]);

  const filteredTrades = useMemo(() => {
    const keyword = searchKeyword.trim().toLowerCase();

    return trades.filter((trade) => {
      const directionMatch =
        directionFilter === "all" || trade.direction === directionFilter;
      if (!directionMatch) {
        return false;
      }

      if (keyword === "") {
        return true;
      }

      const haystack = [
        trade.date,
        trade.direction,
        trade.entryReason,
        trade.tradeType,
        trade.exitReason,
        trade.errorReason,
        trade.note,
      ]
        .join(" ")
        .toLowerCase();

      return haystack.includes(keyword);
    });
  }, [directionFilter, searchKeyword, trades]);

  const applyDashboard = (source: main.TradeDashboard) => {
    const normalized = main.TradeDashboard.createFrom(source);
    setDashboard(normalized);
    setImportPath((prev) => prev || normalized.defaultImportPath || "");
    setExportPath((prev) =>
      prev ||
      (normalized.defaultExportDir
        ? `${normalized.defaultExportDir}/trade-logs-export`
        : "")
    );
    setMessage(
      `已加载 交易:${normalized.trades.length} 下拉项:${normalized.errorTypes.length} 日记:${normalized.journals.length}`
    );
  };

  const refreshDashboard = async () => {
    setLoading(true);
    try {
      const result = await GetDashboard();
      applyDashboard(result);
    } catch (error) {
      setMessage(`加载失败：${String(error)}`);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void refreshDashboard();
  }, []);

  const updateTradeForm = (key: keyof TradeForm, value: string | number) => {
    setTradeForm((prev) => ({
      ...prev,
      [key]: value,
    }));
  };

  const submitTrade = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setLoading(true);

    const payload = new main.TradeEntry({
      id: tradeForm.id,
      date: tradeForm.date,
      note: tradeForm.note,
      entryReason: tradeForm.entryReason,
      tradeType: tradeForm.tradeType,
      exitReason: tradeForm.exitReason,
      supplement: tradeForm.supplement,
      positionSize: parseNumber(tradeForm.positionSize),
      direction: tradeForm.direction,
      entryPrice: parseNumber(tradeForm.entryPrice),
      exitPrice1: parseNumber(tradeForm.exitPrice1),
      exitPrice2: parseNumber(tradeForm.exitPrice2),
      pnl: parseNumber(tradeForm.pnl),
      errorReason: tradeForm.errorReason,
      createdAt: tradeForm.createdAt,
      updatedAt: 0,
    });

    try {
      const result = await SaveTrade(payload);
      applyDashboard(result);
      setTradeForm(emptyTradeForm());
    } catch (error) {
      setMessage(`保存交易失败：${String(error)}`);
    } finally {
      setLoading(false);
    }
  };

  const editTrade = (trade: main.TradeEntry) => {
    setTradeForm({
      id: trade.id,
      date: trade.date || todayString(),
      note: trade.note || "",
      entryReason: trade.entryReason || "",
      tradeType: trade.tradeType || "",
      exitReason: trade.exitReason || "",
      supplement: trade.supplement || "",
      positionSize: trade.positionSize ? String(trade.positionSize) : "1",
      direction: trade.direction || "多",
      entryPrice: trade.entryPrice ? String(trade.entryPrice) : "",
      exitPrice1: trade.exitPrice1 ? String(trade.exitPrice1) : "",
      exitPrice2: trade.exitPrice2 ? String(trade.exitPrice2) : "",
      pnl: trade.pnl ? String(trade.pnl) : "",
      errorReason: trade.errorReason || "",
      createdAt: trade.createdAt || 0,
    });
    setActiveTab("trades");
    setMessage(`正在编辑交易：${trade.date} ${trade.direction || ""}`);
  };

  const removeTrade = (id: string) => {
    const cleanID = (id || "").trim();
    if (cleanID === "") {
      setMessage("删除交易失败：记录ID为空");
      return;
    }
    setConfirmState({
      open: true,
      title: "确认删除交易",
      detail: "删除后无法恢复。",
      target: { kind: "trade", id: cleanID },
    });
  };

  const addErrorOption = async () => {
    const label = newErrorOption.trim();
    if (!label) {
      return;
    }
    setLoading(true);
    try {
      const result = await SaveErrorType(
        new main.ErrorTypeEntry({
          id: "",
          reason: label,
          count: 0,
          exitReason: "",
          updatedAt: 0,
        })
      );
      applyDashboard(result);
      setNewErrorOption("");
    } catch (error) {
      setMessage(`添加错误原因失败：${String(error)}`);
    } finally {
      setLoading(false);
    }
  };

  const addExitOption = async () => {
    const label = newExitOption.trim();
    if (!label) {
      return;
    }
    setLoading(true);
    try {
      const result = await SaveErrorType(
        new main.ErrorTypeEntry({
          id: "",
          reason: "",
          count: 0,
          exitReason: label,
          updatedAt: 0,
        })
      );
      applyDashboard(result);
      setNewExitOption("");
    } catch (error) {
      setMessage(`添加离场理由失败：${String(error)}`);
    } finally {
      setLoading(false);
    }
  };

  const removeErrorOption = (item: main.ErrorTypeEntry) => {
    setConfirmState({
      open: true,
      title: "确认删除错误原因",
      detail: item.reason || "删除后无法恢复。",
      target: { kind: "errorOption", item },
    });
  };

  const removeExitOption = (item: main.ErrorTypeEntry) => {
    setConfirmState({
      open: true,
      title: "确认删除离场理由",
      detail: item.exitReason || "删除后无法恢复。",
      target: { kind: "exitOption", item },
    });
  };

  const submitJournal = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setLoading(true);
    const payload = new main.DailyJournalEntry({
      id: journalForm.id,
      date: journalForm.date,
      ruleExecuted: journalForm.ruleExecuted,
      moodStable: journalForm.moodStable,
      didRecord: journalForm.didRecord,
      prepared: journalForm.prepared,
      noFOMO: journalForm.noFOMO,
      totalPnL: parseNumber(journalForm.totalPnL),
      note: journalForm.note,
      createdAt: journalForm.createdAt,
      updatedAt: 0,
    });
    try {
      const result = await SaveJournal(payload);
      applyDashboard(result);
      setJournalForm(emptyJournalForm());
    } catch (error) {
      setMessage(`保存日记失败：${String(error)}`);
    } finally {
      setLoading(false);
    }
  };

  const editJournal = (item: main.DailyJournalEntry) => {
    setJournalForm({
      id: item.id,
      date: item.date || todayString(),
      ruleExecuted: item.ruleExecuted || "✅",
      moodStable: item.moodStable || "✅",
      didRecord: item.didRecord || "✅",
      prepared: item.prepared || "✅",
      noFOMO: item.noFOMO || "✅",
      totalPnL: String(item.totalPnL ?? 0),
      note: item.note || "",
      createdAt: item.createdAt || 0,
    });
    setActiveTab("journals");
  };

  const removeJournal = (id: string) => {
    const cleanID = (id || "").trim();
    if (cleanID === "") {
      setMessage("删除日记失败：记录ID为空");
      return;
    }
    setConfirmState({
      open: true,
      title: "确认删除日记",
      detail: "删除后无法恢复。",
      target: { kind: "journal", id: cleanID },
    });
  };

  const importFromExcel = async () => {
    setLoading(true);
    try {
      const result = await ImportTradesFromExcel(importPath);
      applyDashboard(result);
      setTradeForm(emptyTradeForm());
      setJournalForm(emptyJournalForm());
      const loaded = main.TradeDashboard.createFrom(result);
      setMessage(
        `导入完成：交易 ${loaded.trades.length} / 下拉项 ${loaded.errorTypes.length} / 日记 ${loaded.journals.length}`
      );
    } catch (error) {
      setMessage(`导入失败：${String(error)}`);
    } finally {
      setLoading(false);
    }
  };

  const exportCsv = async () => {
    setLoading(true);
    try {
      const target = await ExportTradesToCSV(exportPath);
      setMessage(`CSV 导出成功：${target}`);
    } catch (error) {
      setMessage(`CSV 导出失败：${String(error)}`);
    } finally {
      setLoading(false);
    }
  };

  const exportExcel = async () => {
    setLoading(true);
    try {
      const target = await ExportDataToExcel(exportPath);
      setMessage(`Excel 导出成功：${target}`);
    } catch (error) {
      setMessage(`Excel 导出失败：${String(error)}`);
    } finally {
      setLoading(false);
    }
  };

  const cancelConfirm = () => {
    setConfirmState({
      open: false,
      title: "",
      detail: "",
      target: null,
    });
  };

  const confirmDelete = async () => {
    const target = confirmState.target;
    if (!target) {
      cancelConfirm();
      return;
    }

    cancelConfirm();
    setLoading(true);
    try {
      switch (target.kind) {
        case "trade": {
          const result = await DeleteTrade(target.id);
          applyDashboard(result);
          if (tradeForm.id === target.id) {
            setTradeForm(emptyTradeForm());
          }
          setMessage("交易已删除");
          break;
        }
        case "journal": {
          const result = await DeleteJournal(target.id);
          applyDashboard(result);
          if (journalForm.id === target.id) {
            setJournalForm(emptyJournalForm());
          }
          setMessage("日记已删除");
          break;
        }
        case "errorOption": {
          const item = target.item;
          let result: main.TradeDashboard;
          if ((item.exitReason || "").trim() !== "") {
            result = await SaveErrorType(
              new main.ErrorTypeEntry({
                id: item.id,
                reason: "",
                count: item.count,
                exitReason: item.exitReason,
                updatedAt: item.updatedAt,
              })
            );
          } else {
            result = await DeleteErrorType(item.id);
          }
          applyDashboard(result);
          if (tradeForm.errorReason === item.reason) {
            setTradeForm((prev) => ({ ...prev, errorReason: "" }));
          }
          setMessage("错误原因已删除");
          break;
        }
        case "exitOption": {
          const item = target.item;
          let result: main.TradeDashboard;
          if ((item.reason || "").trim() !== "") {
            result = await SaveErrorType(
              new main.ErrorTypeEntry({
                id: item.id,
                reason: item.reason,
                count: item.count,
                exitReason: "",
                updatedAt: item.updatedAt,
              })
            );
          } else {
            result = await DeleteErrorType(item.id);
          }
          applyDashboard(result);
          if (tradeForm.exitReason === item.exitReason) {
            setTradeForm((prev) => ({ ...prev, exitReason: "" }));
          }
          setMessage("离场理由已删除");
          break;
        }
      }
    } catch (error) {
      setMessage(`删除失败：${String(error)}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto flex h-full max-w-[1600px] flex-col gap-5 px-4 py-5 text-slate-800 sm:px-6">
      <header className="panel p-2">
        <nav className="mx-auto flex w-fit gap-1 rounded-sm border border-slate-200 bg-slate-50 p-1">
          <button
            className={`rounded-sm px-4 py-2 text-sm font-semibold transition ${
              activeTab === "trades" ? "bg-ink text-white" : "text-slate-700 hover:bg-slate-100"
            }`}
            onClick={() => setActiveTab("trades")}
            type="button"
          >
            交易日志
          </button>
          <button
            className={`rounded-sm px-4 py-2 text-sm font-semibold transition ${
              activeTab === "journals" ? "bg-ink text-white" : "text-slate-700 hover:bg-slate-100"
            }`}
            onClick={() => setActiveTab("journals")}
            type="button"
          >
            别瞎搞日记本
          </button>
          <button
            className={`rounded-sm px-4 py-2 text-sm font-semibold transition ${
              activeTab === "settings" ? "bg-ink text-white" : "text-slate-700 hover:bg-slate-100"
            }`}
            onClick={() => setActiveTab("settings")}
            type="button"
          >
            设置
          </button>
        </nav>
      </header>

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
        <article className="panel p-4">
          <p className="text-xs uppercase tracking-wide text-slate-500">总交易</p>
          <p className="mt-2 text-2xl font-semibold text-ink">{summary.totalTrades || 0}</p>
        </article>
        <article className="panel p-4">
          <p className="text-xs uppercase tracking-wide text-slate-500">总盈亏</p>
          <p className={`mt-2 text-2xl font-semibold ${summary.totalPnL >= 0 ? "text-accent" : "text-danger"}`}>
            {formatAmount(summary.totalPnL)}
          </p>
        </article>
        <article className="panel p-4">
          <p className="text-xs uppercase tracking-wide text-slate-500">胜率</p>
          <p className="mt-2 text-2xl font-semibold text-ink">{formatAmount(summary.winRate)}%</p>
        </article>
        <article className="panel p-4">
          <p className="text-xs uppercase tracking-wide text-slate-500">平均盈利</p>
          <p className="mt-2 text-2xl font-semibold text-accent">{formatAmount(summary.avgWin)}</p>
        </article>
        <article className="panel p-4">
          <p className="text-xs uppercase tracking-wide text-slate-500">平均亏损</p>
          <p className="mt-2 text-2xl font-semibold text-danger">{formatAmount(summary.avgLoss)}</p>
        </article>
        <article className="panel p-4">
          <p className="text-xs uppercase tracking-wide text-slate-500">下拉项总数</p>
          <p className="mt-2 text-2xl font-semibold text-ink">{errorTypes.length}</p>
        </article>
      </section>

      {activeTab === "trades" && (
        <section className="grid gap-4 xl:grid-cols-[430px_1fr]">
          <form className="panel grid gap-3 p-4" onSubmit={submitTrade}>
            <h2 className="text-lg font-semibold text-ink">{tradeForm.id ? "编辑交易" : "新增交易"}</h2>
            <div className="grid grid-cols-2 gap-2">
              <input
                type="date"
                value={tradeForm.date}
                className="input-base"
                onChange={(event) => updateTradeForm("date", event.target.value)}
                required
              />
              <CustomSelect
                value={tradeForm.direction}
                options={directionOptions}
                onChange={(next) => updateTradeForm("direction", next)}
              />
            </div>

            <div className="grid grid-cols-2 gap-2">
              <input
                className="input-base"
                value={tradeForm.positionSize}
                onChange={(event) => updateTradeForm("positionSize", event.target.value)}
                placeholder="仓位大小"
              />
              <input
                className="input-base"
                value={tradeForm.tradeType}
                onChange={(event) => updateTradeForm("tradeType", event.target.value)}
                placeholder="类型"
              />
            </div>

            <div className="grid grid-cols-3 gap-2">
              <input
                className="input-base"
                value={tradeForm.entryPrice}
                onChange={(event) => updateTradeForm("entryPrice", event.target.value)}
                placeholder="入场价"
              />
              <input
                className="input-base"
                value={tradeForm.exitPrice1}
                onChange={(event) => updateTradeForm("exitPrice1", event.target.value)}
                placeholder="离场价1"
              />
              <input
                className="input-base"
                value={tradeForm.exitPrice2}
                onChange={(event) => updateTradeForm("exitPrice2", event.target.value)}
                placeholder="离场价2"
              />
            </div>

            <input
              className="input-base"
              value={tradeForm.pnl}
              onChange={(event) => updateTradeForm("pnl", event.target.value)}
              placeholder="盈亏（留空则自动计算）"
            />
            <input
              className="input-base"
              value={tradeForm.entryReason}
              onChange={(event) => updateTradeForm("entryReason", event.target.value)}
              placeholder="入场理由"
            />

            <div className="grid grid-cols-2 gap-2">
              <CustomSelect
                value={tradeForm.exitReason}
                placeholder="选择离场理由"
                searchable
                searchPlaceholder="搜索离场理由"
                options={[
                  { value: "", label: "选择离场理由" },
                  ...exitOptionRows.map((item) => ({
                    value: item.exitReason,
                    label: item.exitReason,
                  })),
                ]}
                onChange={(next) => updateTradeForm("exitReason", next)}
              />

              <CustomSelect
                value={tradeForm.errorReason}
                placeholder="选择错误原因"
                searchable
                searchPlaceholder="搜索错误原因"
                options={[
                  { value: "", label: "选择错误原因" },
                  ...errorOptionRows.map((item) => ({
                    value: item.reason,
                    label: item.reason,
                  })),
                ]}
                onChange={(next) => updateTradeForm("errorReason", next)}
              />
            </div>

            <textarea
              className="min-h-[90px] rounded-sm border border-slate-300 bg-white p-3 text-sm text-slate-800 outline-none transition focus:border-slate-700 focus:ring-2 focus:ring-slate-200"
              value={tradeForm.note}
              onChange={(event) => updateTradeForm("note", event.target.value)}
              placeholder="备注"
            />
            <textarea
              className="min-h-[70px] rounded-sm border border-slate-300 bg-white p-3 text-sm text-slate-800 outline-none transition focus:border-slate-700 focus:ring-2 focus:ring-slate-200"
              value={tradeForm.supplement}
              onChange={(event) => updateTradeForm("supplement", event.target.value)}
              placeholder="补充说明"
            />

            <div className="flex gap-2">
              <button
                type="submit"
                className="h-10 flex-1 rounded-sm bg-ink text-sm font-semibold text-white transition hover:bg-slate-900 disabled:opacity-60"
                disabled={loading}
              >
                {tradeForm.id ? "保存修改" : "添加交易"}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setTradeForm(emptyTradeForm())}>
                清空
              </button>
            </div>
          </form>

          <div className="panel flex min-h-[360px] flex-col overflow-hidden">
            <div className="flex flex-col gap-2 border-b border-slate-200 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex gap-2">
                <CustomSelect
                  className="w-[120px]"
                  value={directionFilter}
                  options={directionFilterOptions}
                  onChange={(next) => setDirectionFilter(next as DirectionFilter)}
                />
                <input
                  className="input-base w-[260px]"
                  value={searchKeyword}
                  onChange={(event) => setSearchKeyword(event.target.value)}
                  placeholder="搜索理由 / 错误 / 备注"
                />
              </div>
            </div>

            <div className="overflow-auto">
              <table className="min-w-full border-collapse text-sm">
                <thead className="sticky top-0 bg-slate-100/95 text-left text-xs uppercase tracking-wide text-slate-600">
                  <tr>
                    <th className="px-3 py-2">日期</th>
                    <th className="px-3 py-2">方向</th>
                    <th className="px-3 py-2">入场/离场</th>
                    <th className="px-3 py-2">仓位</th>
                    <th className="px-3 py-2">盈亏</th>
                    <th className="px-3 py-2">理由</th>
                    <th className="px-3 py-2">备注</th>
                    <th className="px-3 py-2">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredTrades.map((trade) => (
                    <tr key={trade.id} className="border-b border-slate-200/70 bg-white/40 hover:bg-white/80">
                      <td className="whitespace-nowrap px-3 py-2">{trade.date || "-"}</td>
                      <td className="px-3 py-2">{trade.direction || "-"}</td>
                      <td className="whitespace-nowrap px-3 py-2">
                        {formatAmount(trade.entryPrice)} / {formatAmount(trade.exitPrice1 || trade.exitPrice2)}
                      </td>
                      <td className="px-3 py-2">{formatAmount(trade.positionSize)}</td>
                      <td
                        className={`px-3 py-2 font-semibold ${
                          trade.pnl > 0 ? "text-accent" : trade.pnl < 0 ? "text-danger" : "text-slate-500"
                        }`}
                      >
                        {formatAmount(trade.pnl)}
                      </td>
                      <td className="max-w-[250px] px-3 py-2">
                        <p className="truncate">{trade.entryReason || "-"}</p>
                        <p className="truncate text-xs text-slate-500">{trade.exitReason || "-"}</p>
                      </td>
                      <td className="max-w-[220px] px-3 py-2 text-xs text-slate-600">
                        {trade.errorReason || trade.note || "-"}
                      </td>
                      <td className="px-3 py-2">
                        <div className="flex gap-2">
                          <button
                            className="rounded-sm border border-slate-300 px-2 py-1 text-xs hover:bg-slate-100"
                            onClick={() => editTrade(trade)}
                            type="button"
                          >
                            编辑
                          </button>
                          <button
                            className="rounded-sm border border-red-200 px-2 py-1 text-xs text-danger hover:bg-red-50"
                            onClick={() => void removeTrade(trade.id)}
                            type="button"
                          >
                            删除
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {filteredTrades.length === 0 && (
                    <tr>
                      <td className="px-4 py-8 text-center text-slate-500" colSpan={8}>
                        暂无符合条件的交易记录
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </section>
      )}

      {activeTab === "journals" && (
        <section className="grid gap-4 xl:grid-cols-[460px_1fr]">
          <form className="panel grid gap-3 p-4" onSubmit={submitJournal}>
            <h2 className="text-lg font-semibold text-ink">{journalForm.id ? "编辑日记" : "新增日记"}</h2>
            <input
              type="date"
              className="input-base"
              value={journalForm.date}
              onChange={(event) => setJournalForm((prev) => ({ ...prev, date: event.target.value }))}
              required
            />

            <div className="grid grid-cols-2 gap-2">
              <CustomSelect
                value={journalForm.ruleExecuted}
                options={yesNoOptions("执行规则 ✅", "执行规则 ❌")}
                onChange={(next) => setJournalForm((prev) => ({ ...prev, ruleExecuted: next }))}
              />
              <CustomSelect
                value={journalForm.moodStable}
                options={yesNoOptions("情绪稳定 ✅", "情绪稳定 ❌")}
                onChange={(next) => setJournalForm((prev) => ({ ...prev, moodStable: next }))}
              />
            </div>

            <div className="grid grid-cols-2 gap-2">
              <CustomSelect
                value={journalForm.didRecord}
                options={yesNoOptions("做记录 ✅", "做记录 ❌")}
                onChange={(next) => setJournalForm((prev) => ({ ...prev, didRecord: next }))}
              />
              <CustomSelect
                value={journalForm.prepared}
                options={yesNoOptions("提前准备 ✅", "提前准备 ❌")}
                onChange={(next) => setJournalForm((prev) => ({ ...prev, prepared: next }))}
              />
            </div>

            <div className="grid grid-cols-2 gap-2">
              <CustomSelect
                value={journalForm.noFOMO}
                options={yesNoOptions("无 FOMO ✅", "无 FOMO ❌")}
                onChange={(next) => setJournalForm((prev) => ({ ...prev, noFOMO: next }))}
              />
              <input
                className="input-base"
                value={journalForm.totalPnL}
                onChange={(event) => setJournalForm((prev) => ({ ...prev, totalPnL: event.target.value }))}
                placeholder="总盈亏"
              />
            </div>

            <textarea
              className="min-h-[90px] rounded-sm border border-slate-300 bg-white p-3 text-sm text-slate-800 outline-none transition focus:border-slate-700 focus:ring-2 focus:ring-slate-200"
              value={journalForm.note}
              onChange={(event) => setJournalForm((prev) => ({ ...prev, note: event.target.value }))}
              placeholder="备注"
            />

            <div className="flex gap-2">
              <button
                type="submit"
                className="h-10 flex-1 rounded-sm bg-ink text-sm font-semibold text-white transition hover:bg-slate-900 disabled:opacity-60"
                disabled={loading}
              >
                {journalForm.id ? "保存修改" : "添加日记"}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setJournalForm(emptyJournalForm())}>
                清空
              </button>
            </div>
          </form>

          <div className="panel overflow-auto">
            <table className="min-w-full border-collapse text-sm">
              <thead className="sticky top-0 bg-slate-100/95 text-left text-xs uppercase tracking-wide text-slate-600">
                <tr>
                  <th className="px-3 py-2">日期</th>
                  <th className="px-3 py-2">规则</th>
                  <th className="px-3 py-2">情绪</th>
                  <th className="px-3 py-2">记录</th>
                  <th className="px-3 py-2">准备</th>
                  <th className="px-3 py-2">FOMO</th>
                  <th className="px-3 py-2">总盈亏</th>
                  <th className="px-3 py-2">备注</th>
                  <th className="px-3 py-2">操作</th>
                </tr>
              </thead>
              <tbody>
                {journals.map((item) => (
                  <tr key={item.id} className="border-b border-slate-200/70 bg-white/40 hover:bg-white/80">
                    <td className="px-3 py-2">{item.date || "-"}</td>
                    <td className="px-3 py-2">{item.ruleExecuted || "-"}</td>
                    <td className="px-3 py-2">{item.moodStable || "-"}</td>
                    <td className="px-3 py-2">{item.didRecord || "-"}</td>
                    <td className="px-3 py-2">{item.prepared || "-"}</td>
                    <td className="px-3 py-2">{item.noFOMO || "-"}</td>
                    <td className={`px-3 py-2 font-semibold ${item.totalPnL >= 0 ? "text-accent" : "text-danger"}`}>
                      {formatAmount(item.totalPnL)}
                    </td>
                    <td className="max-w-[180px] px-3 py-2 text-xs text-slate-600">{item.note || "-"}</td>
                    <td className="px-3 py-2">
                      <div className="flex gap-2">
                        <button
                          className="rounded-sm border border-slate-300 px-2 py-1 text-xs hover:bg-slate-100"
                          onClick={() => editJournal(item)}
                          type="button"
                        >
                          编辑
                        </button>
                        <button
                          className="rounded-sm border border-red-200 px-2 py-1 text-xs text-danger hover:bg-red-50"
                          onClick={() => void removeJournal(item.id)}
                          type="button"
                        >
                          删除
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {journals.length === 0 && (
                  <tr>
                    <td className="px-4 py-8 text-center text-slate-500" colSpan={9}>
                      暂无日记记录
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {activeTab === "settings" && (
        <section className="grid gap-4 xl:grid-cols-[1fr_1fr]">
          <div className="panel p-3 xl:col-span-2">
            <p className="text-xs text-slate-600">{message}</p>
          </div>

          <section className="panel p-4">
            <h2 className="mb-3 text-base font-semibold text-slate-800">数据导入导出</h2>
            <div className="grid gap-2">
              <label className="text-xs font-medium text-slate-600">导入路径（Excel）</label>
              <div className="flex gap-2">
                <input
                  className="input-base flex-1"
                  value={importPath}
                  onChange={(event) => setImportPath(event.target.value)}
                  placeholder="Excel 导入路径"
                />
                <button
                  className="h-10 rounded-sm bg-ink px-4 text-sm font-semibold text-white transition hover:bg-slate-900 disabled:opacity-60"
                  onClick={importFromExcel}
                  disabled={loading}
                  type="button"
                >
                  导入
                </button>
              </div>

              <label className="mt-3 text-xs font-medium text-slate-600">导出路径</label>
              <input
                className="input-base"
                value={exportPath}
                onChange={(event) => setExportPath(event.target.value)}
                placeholder="导出路径（可填完整文件名）"
              />
              <div className="flex gap-2">
                <button className="btn-secondary" onClick={exportCsv} disabled={loading} type="button">
                  导出 CSV
                </button>
                <button className="btn-secondary" onClick={exportExcel} disabled={loading} type="button">
                  导出 Excel
                </button>
              </div>
            </div>
          </section>

          <section className="panel p-4">
            <h2 className="mb-3 text-base font-semibold text-slate-800">交易下拉项配置</h2>
            <div className="grid gap-4 md:grid-cols-2">
              <section>
                <h3 className="mb-2 text-sm font-semibold text-slate-700">错误原因</h3>
                <div className="mb-2 flex gap-2">
                  <input
                    className="input-base flex-1"
                    value={newErrorOption}
                    onChange={(event) => setNewErrorOption(event.target.value)}
                    placeholder="新增错误原因"
                  />
                  <button className="btn-secondary" onClick={() => void addErrorOption()} type="button">
                    添加
                  </button>
                </div>
                <div className="max-h-[280px] overflow-auto border border-slate-200">
                  {errorOptionRows.map((item) => (
                    <div
                      key={`err-${item.id}`}
                      className="flex items-center justify-between border-b border-slate-100 px-3 py-2 text-sm last:border-0"
                    >
                      <span className="truncate pr-2">{item.reason}</span>
                      <button
                        className="rounded-sm border border-red-200 px-2 py-1 text-xs text-danger hover:bg-red-50"
                        onClick={() => void removeErrorOption(item)}
                        type="button"
                      >
                        删除
                      </button>
                    </div>
                  ))}
                  {errorOptionRows.length === 0 && (
                    <div className="px-3 py-6 text-center text-xs text-slate-500">暂无错误原因</div>
                  )}
                </div>
              </section>

              <section>
                <h3 className="mb-2 text-sm font-semibold text-slate-700">离场理由</h3>
                <div className="mb-2 flex gap-2">
                  <input
                    className="input-base flex-1"
                    value={newExitOption}
                    onChange={(event) => setNewExitOption(event.target.value)}
                    placeholder="新增离场理由"
                  />
                  <button className="btn-secondary" onClick={() => void addExitOption()} type="button">
                    添加
                  </button>
                </div>
                <div className="max-h-[280px] overflow-auto border border-slate-200">
                  {exitOptionRows.map((item) => (
                    <div
                      key={`exit-${item.id}`}
                      className="flex items-center justify-between border-b border-slate-100 px-3 py-2 text-sm last:border-0"
                    >
                      <span className="truncate pr-2">{item.exitReason}</span>
                      <button
                        className="rounded-sm border border-red-200 px-2 py-1 text-xs text-danger hover:bg-red-50"
                        onClick={() => void removeExitOption(item)}
                        type="button"
                      >
                        删除
                      </button>
                    </div>
                  ))}
                  {exitOptionRows.length === 0 && (
                    <div className="px-3 py-6 text-center text-xs text-slate-500">暂无离场理由</div>
                  )}
                </div>
              </section>
            </div>
          </section>
        </section>
      )}

      {confirmState.open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/30 p-4">
          <div className="panel w-full max-w-md p-4">
            <h3 className="text-base font-semibold text-slate-800">{confirmState.title}</h3>
            <p className="mt-2 text-sm text-slate-600">{confirmState.detail}</p>
            <div className="mt-4 flex justify-end gap-2">
              <button className="btn-secondary" onClick={cancelConfirm} type="button" disabled={loading}>
                取消
              </button>
              <button
                className="h-10 rounded-sm bg-ink px-4 text-sm font-semibold text-white transition hover:bg-slate-900 disabled:opacity-60"
                onClick={() => void confirmDelete()}
                type="button"
                disabled={loading}
              >
                确认删除
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
