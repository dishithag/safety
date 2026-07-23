import {
  lazy,
  Suspense,
  useDeferredValue,
  useEffect,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { ReportMarkdown, reportTableOfContents } from "./components/ReportMarkdown";
import {
  formatObjectSize,
  narrativeCatalogURL,
  parseNarrativeCatalog,
  type CatalogReport,
} from "./data/narrativeCatalog";
import { compactCID } from "./data/reports";

type AppView = "reports" | "analytics";

const AnalyticsDashboard = lazy(() =>
  import("./components/AnalyticsDashboard").then((module) => ({
    default: module.AnalyticsDashboard,
  })),
);

type NarrativeState =
  | { status: "idle"; markdown: ""; message: "" }
  | { status: "loading"; markdown: ""; message: "" }
  | { status: "ready"; markdown: string; message: "" }
  | { status: "not-found" | "error"; markdown: ""; message: string };

type CatalogState = {
  status: "loading" | "refreshing" | "ready" | "error";
  reports: CatalogReport[];
  message: string;
};

const apiBaseURL = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");

function narrativeURL(cid: string): string {
  return `${apiBaseURL}/zero-trust-analytics/narratives/${encodeURIComponent(cid)}`;
}

function reportMatches(report: CatalogReport, query: string): boolean {
  const normalized = query.trim().toLowerCase();
  if (normalized === "") {
    return true;
  }
  return (
    report.cid.toLowerCase().includes(normalized) ||
    report.platforms.some((platform) => platform.toLowerCase().includes(normalized))
  );
}

export default function App() {
  const [activeView, setActiveView] = useState<AppView>("reports");
  const [selectedCID, setSelectedCID] = useState("");
  const [query, setQuery] = useState("");
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [reloadVersion, setReloadVersion] = useState(0);
  const [catalogReloadVersion, setCatalogReloadVersion] = useState(0);
  const [catalog, setCatalog] = useState<CatalogState>({
    status: "loading",
    reports: [],
    message: "",
  });
  const directLookup = useRef(false);
  const [narrative, setNarrative] = useState<NarrativeState>({
    status: "idle",
    markdown: "",
    message: "",
  });
  const deferredQuery = useDeferredValue(query);
  const selectedReport = catalog.reports.find((report) => report.cid === selectedCID);
  const matchingReports = catalog.reports.filter((report) =>
    reportMatches(report, deferredQuery),
  );
  const selectedRevision = selectedReport?.last_modified ?? "";
  const tableOfContents =
    narrative.status === "ready" ? reportTableOfContents(narrative.markdown) : [];

  useEffect(() => {
    const controller = new AbortController();
    setCatalog((current) => ({
      status: current.reports.length > 0 ? "refreshing" : "loading",
      reports: current.reports,
      message: "",
    }));

    async function loadCatalog() {
      try {
        const response = await fetch(narrativeCatalogURL, {
          cache: "no-store",
          signal: controller.signal,
          headers: { Accept: "application/json" },
        });
        if (!response.ok) {
          throw new Error(`The demo catalog returned HTTP ${response.status}.`);
        }

        const reports = parseNarrativeCatalog(await response.json());
        setCatalog({ status: "ready", reports, message: "" });
        setSelectedCID((current) => {
          if (current === "") {
            directLookup.current = false;
            return reports[0]?.cid ?? "";
          }
          if (reports.some((report) => report.cid === current)) {
            directLookup.current = false;
            return current;
          }
          if (directLookup.current) {
            return current;
          }
          return reports[0]?.cid ?? "";
        });
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        setCatalog((current) => ({
          status: "error",
          reports: current.reports,
          message:
            error instanceof Error
              ? error.message
              : "The narrative catalog could not be loaded from MinIO.",
        }));
      }
    }

    void loadCatalog();
    return () => controller.abort();
  }, [catalogReloadVersion]);

  useEffect(() => {
    const refresh = () => setCatalogReloadVersion((version) => version + 1);
    const interval = window.setInterval(refresh, 15_000);
    window.addEventListener("focus", refresh);
    return () => {
      window.clearInterval(interval);
      window.removeEventListener("focus", refresh);
    };
  }, []);

  useEffect(() => {
    if (selectedCID === "") {
      setNarrative({ status: "idle", markdown: "", message: "" });
      return;
    }

    const controller = new AbortController();
    setNarrative({ status: "loading", markdown: "", message: "" });
    setCopied(false);

    async function loadNarrative() {
      try {
        const response = await fetch(narrativeURL(selectedCID), {
          cache: "no-store",
          signal: controller.signal,
          headers: { Accept: "text/markdown, text/plain" },
        });
        if (response.status === 404) {
          setNarrative({
            status: "not-found",
            markdown: "",
            message: "No generated narrative exists for this CID.",
          });
          return;
        }
        if (!response.ok) {
          throw new Error(`The API returned HTTP ${response.status}.`);
        }

        const markdown = await response.text();
        if (markdown.trim() === "") {
          throw new Error("The API returned an empty narrative.");
        }
        setNarrative({ status: "ready", markdown, message: "" });
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        setNarrative({
          status: "error",
          markdown: "",
          message:
            error instanceof Error
              ? error.message
              : "The narrative could not be loaded from analyticsapi.",
        });
      }
    }

    void loadNarrative();
    return () => controller.abort();
  }, [selectedCID, selectedRevision, reloadVersion]);

  function selectReport(cid: string) {
    directLookup.current = false;
    setSelectedCID(cid);
    if (cid === selectedCID) {
      setReloadVersion((version) => version + 1);
    }
    setSidebarOpen(false);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  function changeView(view: AppView) {
    if (view === "reports" && activeView !== "reports") {
      setCatalogReloadVersion((version) => version + 1);
    }
    setActiveView(view);
    setSidebarOpen(false);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  async function copyCID() {
    await navigator.clipboard.writeText(selectedCID);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  }

  function requestCID(cid: string) {
    const requestedCID = cid.trim();
    if (requestedCID === "") {
      return;
    }
    directLookup.current = true;
    setSelectedCID(requestedCID);
    setReloadVersion((version) => version + 1);
    setSidebarOpen(false);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  function openCID(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    requestCID(query);
  }

  return (
    <div className="app-shell">
      <header className="app-header">
        {activeView === "reports" && (
          <button
            className="sidebar-toggle"
            type="button"
            aria-label="Open report list"
            aria-expanded={sidebarOpen}
            onClick={() => setSidebarOpen((open) => !open)}
          >
            <span />
            <span />
            <span />
          </button>
        )}

        <div className="brand-mark" aria-hidden="true">
          <span>Z</span>
        </div>
        <div className="brand-copy">
          <p>Security posture workspace</p>
          <h1>Zero Trust Assessment Reports</h1>
        </div>
        <nav className="primary-nav" aria-label="Demo workspace">
          <button
            type="button"
            className={activeView === "reports" ? "active" : ""}
            aria-current={activeView === "reports" ? "page" : undefined}
            onClick={() => changeView("reports")}
          >
            Reports
          </button>
          <button
            type="button"
            className={activeView === "analytics" ? "active" : ""}
            aria-current={activeView === "analytics" ? "page" : undefined}
            onClick={() => changeView("analytics")}
          >
            Analytics
          </button>
        </nav>
        <div className="demo-label">
          <span /> Demo environment
        </div>
      </header>

      {activeView === "reports" ? (
        <div className="workspace">
        {sidebarOpen && (
          <button
            className="sidebar-scrim"
            aria-label="Close report list"
            type="button"
            onClick={() => setSidebarOpen(false)}
          />
        )}

        <aside className={`report-sidebar ${sidebarOpen ? "report-sidebar--open" : ""}`}>
          <div className="sidebar-heading">
            <div>
              <p className="eyebrow">Narrative library</p>
              <h2>Customer reports</h2>
            </div>
            <span className="report-count" title="Narratives currently stored in MinIO">
              {catalog.reports.length}
            </span>
          </div>

          <form className="search-field cid-lookup" onSubmit={openCID}>
            <span className="search-icon" aria-hidden="true" />
            <label className="sr-only" htmlFor="cid-search">Search reports or open a CID</label>
            <input
              id="cid-search"
              type="search"
              placeholder="Search or enter a CID"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <button type="submit" disabled={query.trim() === ""}>Open</button>
          </form>

          <nav className="report-list" aria-label="Available CID reports">
            {catalog.status === "loading" && catalog.reports.length === 0 && (
              <CatalogLoading />
            )}
            {catalog.status === "error" && (
              <div className="catalog-alert" role="alert">
                <strong>Could not sync MinIO</strong>
                <span>{catalog.message}</span>
                <button
                  type="button"
                  onClick={() => setCatalogReloadVersion((version) => version + 1)}
                >
                  Retry
                </button>
              </div>
            )}
            {matchingReports.map((report, index) => {
              const selected = report.cid === selectedCID;
              return (
                <button
                  key={report.cid}
                  className={`report-list-item ${selected ? "report-list-item--selected" : ""}`}
                  type="button"
                  aria-current={selected ? "page" : undefined}
                  onClick={() => selectReport(report.cid)}
                >
                  <span className="report-index">{String(index + 1).padStart(2, "0")}</span>
                  <span className="report-list-copy">
                    <strong>{compactCID(report.cid)}</strong>
                    <small>
                      {report.platforms.length > 0 ? (
                        <>
                          {report.platforms.length}{" "}
                          {report.platforms.length === 1 ? "platform" : "platforms"}
                          <span aria-hidden="true"> / </span>
                          {report.platforms.slice(0, 2).join(", ")}
                          {report.platforms.length > 2
                            ? ` +${report.platforms.length - 2}`
                            : ""}
                        </>
                      ) : (
                        <>Generated narrative / {formatObjectSize(report.size_bytes)}</>
                      )}
                    </small>
                  </span>
                </button>
              );
            })}
            {catalog.status === "ready" && catalog.reports.length === 0 && (
              <div className="empty-search">
                <strong>No generated narratives</strong>
                <span>The sidebar will populate when Markdown summaries are written.</span>
              </div>
            )}
            {catalog.reports.length > 0 && matchingReports.length === 0 && (
              <div className="empty-search">
                <strong>No matching shortcut</strong>
                <span>Open the entered CID directly to check whether its narrative exists.</span>
                <button type="button" onClick={() => requestCID(query)}>Open CID</button>
              </div>
            )}
          </nav>
        </aside>

        <main className="report-workspace">
          {selectedCID !== "" ? (
            <>
              <section className="report-toolbar" aria-label="Selected report details">
                <div>
                  <p className="eyebrow">Selected customer identifier</p>
                  <div className="cid-line">
                    <code>{selectedCID}</code>
                    <button className="copy-button" type="button" onClick={() => void copyCID()}>
                      {copied ? "Copied" : "Copy CID"}
                    </button>
                  </div>
                </div>
                <div className="toolbar-meta">
                  <span className={`load-status load-status--${narrative.status}`}>
                    <i />
                    {narrative.status === "ready" && "Narrative ready"}
                    {narrative.status === "loading" && "Loading narrative"}
                    {narrative.status === "not-found" && "Not generated"}
                    {narrative.status === "error" && "API unavailable"}
                  </span>
                  <span className="platform-count">
                    {selectedReport && selectedReport.platforms.length > 0
                      ? `${selectedReport.platforms.length} ${selectedReport.platforms.length === 1 ? "platform" : "platforms"}`
                      : selectedReport
                        ? "Generated narrative"
                        : "Direct lookup"}
                  </span>
                </div>
              </section>

              <div className="report-layout">
                <article className="report-paper">
                  {narrative.status === "loading" && <LoadingReport />}
                  {(narrative.status === "error" || narrative.status === "not-found") && (
                    <ReportError
                      message={narrative.message}
                      status={narrative.status}
                      onRetry={() => setReloadVersion((version) => version + 1)}
                    />
                  )}
                  {narrative.status === "ready" && (
                    <div className="markdown-body">
                      <ReportMarkdown markdown={narrative.markdown} />
                    </div>
                  )}
                </article>

                {narrative.status === "ready" && tableOfContents.length > 0 && (
                  <aside className="document-outline" aria-label="Report contents">
                    <p className="eyebrow">On this report</p>
                    <nav>
                      {tableOfContents.map((item) => (
                        <a
                          key={item.id}
                          className={item.depth === 3 ? "outline-child" : ""}
                          href={`#${item.id}`}
                        >
                          {item.label}
                        </a>
                      ))}
                    </nav>
                  </aside>
                )}
              </div>
            </>
          ) : (
            <div className="report-layout report-layout--empty">
              <article className="report-paper">
                {catalog.status === "loading" ? (
                  <LoadingReport />
                ) : (
                  <EmptyCatalog
                    isError={catalog.status === "error"}
                    message={catalog.message}
                    onRetry={() => setCatalogReloadVersion((version) => version + 1)}
                  />
                )}
              </article>
            </div>
          )}
        </main>
        </div>
      ) : (
        <Suspense fallback={<AnalyticsLoading />}>
          <AnalyticsDashboard />
        </Suspense>
      )}
    </div>
  );
}

function AnalyticsLoading() {
  return (
    <main className="analytics-workspace analytics-loading" aria-label="Loading analytics dashboard">
      <div className="analytics-loading-heading" />
      <div className="metric-grid">
        {[0, 1, 2, 3].map((item) => (
          <div className="analytics-loading-card" key={item} />
        ))}
      </div>
      <div className="analytics-loading-chart" />
    </main>
  );
}

function CatalogLoading() {
  return (
    <div className="catalog-loading" aria-live="polite" aria-label="Loading MinIO narratives">
      {[0, 1, 2, 3].map((item) => (
        <div className="catalog-loading-row" key={item}>
          <span />
          <i />
        </div>
      ))}
    </div>
  );
}

type EmptyCatalogProps = {
  isError: boolean;
  message: string;
  onRetry: () => void;
};

function EmptyCatalog({ isError, message, onRetry }: EmptyCatalogProps) {
  return (
    <div className="report-error" role={isError ? "alert" : "status"}>
      <div className="error-glyph" aria-hidden="true">{isError ? "!" : "0"}</div>
      <p className="eyebrow">{isError ? "MinIO unavailable" : "Narrative library"}</p>
      <h2>{isError ? "Could not load generated reports" : "No narratives are available"}</h2>
      <p>
        {isError
          ? message
          : "The sidebar will populate automatically after the summarizer writes Markdown files."}
      </p>
      <button type="button" onClick={onRetry}>Retry MinIO</button>
    </div>
  );
}

function LoadingReport() {
  return (
    <div className="loading-report" aria-live="polite" aria-label="Loading narrative">
      <div className="loading-kicker" />
      <div className="loading-title" />
      <div className="loading-meta" />
      <div className="loading-rule" />
      {[92, 76, 88, 62, 84, 71].map((width, index) => (
        <div key={`${width}-${index}`} className="loading-line" style={{ width: `${width}%` }} />
      ))}
    </div>
  );
}

type ReportErrorProps = {
  message: string;
  status: "error" | "not-found";
  onRetry: () => void;
};

function ReportError({ message, status, onRetry }: ReportErrorProps) {
  return (
    <div className="report-error" role="alert">
      <div className="error-glyph" aria-hidden="true">
        {status === "not-found" ? "404" : "!"}
      </div>
      <p className="eyebrow">{status === "not-found" ? "Narrative unavailable" : "Connection problem"}</p>
      <h2>{status === "not-found" ? "This report has not been generated" : "Could not reach analyticsapi"}</h2>
      <p>{message}</p>
      <button type="button" onClick={onRetry}>
        Try again
      </button>
    </div>
  );
}
