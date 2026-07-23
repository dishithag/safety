import { useState } from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  ReferenceLine,
  ResponsiveContainer,
  Scatter,
  ScatterChart,
  Tooltip,
  XAxis,
  YAxis,
  ZAxis,
} from "recharts";
import {
  cidControlMovers,
  cidPlatformTrendSeries,
  cidSummary,
  cidTrendSeries,
  cidZeroTransitions,
  compactTrendCID,
  platformComparison,
  portfolioCIDMovers,
  portfolioScatterData,
  portfolioScoreDistribution,
  portfolioSummary,
  portfolioTrendSeries,
  recurringControlGaps,
  trendFixture,
  zeroComplianceControls,
  type CIDHistory,
  type ZeroControlTransition,
  type ZeroTransitionStatus,
} from "../data/trends";

type AnalyticsView = "portfolio" | "cid";

const chartColors = {
  overall: "#d71936",
  os: "#1c3038",
  sensor: "#0d806d",
  weighted: "#c57714",
  baseline: "#b9b6ae",
  positive: "#0d806d",
  negative: "#d71936",
  unchanged: "#98958e",
};

const scoreBandColors = ["#9f1026", "#d34a2c", "#c98a20", "#4f9787", "#176a5e"];
const platformColors = ["#d71936", "#0d806d", "#c57714", "#315d76", "#8c5b91", "#68733c"];
const transitionLabels: Record<ZeroTransitionStatus, string> = {
  resolved: "Resolved",
  introduced: "Introduced",
  unchanged: "Still zero",
};

const transitionColors: Record<ZeroTransitionStatus, string> = {
  resolved: chartColors.positive,
  introduced: chartColors.negative,
  unchanged: chartColors.unchanged,
};

function score(value: number | null, digits = 1): string {
  return value == null ? "N/A" : value.toFixed(digits);
}

function signed(value: number | null, suffix = ""): string {
  if (value == null) {
    return "N/A";
  }
  const prefix = value > 0 ? "+" : "";
  return `${prefix}${value.toFixed(1)}${suffix}`;
}

function devices(value: number): string {
  return new Intl.NumberFormat("en-US", { notation: "compact", maximumFractionDigits: 1 }).format(value);
}

function directionClass(value: number | null): string {
  if (value == null || value === 0) {
    return "metric-neutral";
  }
  return value > 0 ? "metric-positive" : "metric-negative";
}

function transitionPreview(transitions: ZeroControlTransition[]): string {
  if (transitions.length === 0) {
    return "No controls";
  }
  const first = transitions[0];
  return transitions.length === 1
    ? `${first.label} / ${first.platform}`
    : `${first.label} +${transitions.length - 1} more`;
}

type MetricCardProps = {
  label: string;
  value: string;
  detail: string;
  direction?: number | null;
};

function MetricCard({ label, value, detail, direction = null }: MetricCardProps) {
  return (
    <article className="metric-card">
      <p>{label}</p>
      <strong>{value}</strong>
      <span className={directionClass(direction)}>{detail}</span>
    </article>
  );
}

type ChartPanelProps = {
  title: string;
  description: string;
  className?: string;
  children: React.ReactNode;
};

function ChartPanel({ title, description, className = "", children }: ChartPanelProps) {
  return (
    <section className={`analytics-panel ${className}`}>
      <header className="panel-heading">
        <div>
          <h3>{title}</h3>
          <p>{description}</p>
        </div>
      </header>
      {children}
    </section>
  );
}

function ChartTooltipCard({ active, payload, label }: {
  active?: boolean;
  payload?: ReadonlyArray<{
    name?: string;
    value?: number | string;
    color?: string;
  }>;
  label?: string;
}) {
  if (!active || !payload?.length) {
    return null;
  }
  return (
    <div className="chart-tooltip">
      {label && <strong>{label}</strong>}
      {payload.map((item) => (
        <span key={`${item.name}-${item.value}`}>
          <i style={{ background: item.color }} />
          {item.name}: <b>{typeof item.value === "number" ? item.value.toFixed(1) : item.value}</b>
        </span>
      ))}
    </div>
  );
}

export function AnalyticsDashboard() {
  const [view, setView] = useState<AnalyticsView>("portfolio");
  const [selectedCID, setSelectedCID] = useState(trendFixture.cids[0].cid);
  const selectedHistory =
    trendFixture.cids.find((history) => history.cid === selectedCID) ?? trendFixture.cids[0];

  return (
    <main className="analytics-workspace">
      <section className="analytics-hero">
        <div>
          <p className="eyebrow">Future-scope demonstration</p>
          <h2>Seven-day posture intelligence</h2>
          <p className="analytics-intro">
            Compare portfolio direction, recurring control gaps, and daily movement without
            changing the production reporting pipeline.
          </p>
        </div>
        <div className="synthetic-badge">
          <span>Synthetic history</span>
          {trendFixture.metadata.period_start} to {trendFixture.metadata.period_end}
        </div>
      </section>

      <section className="analytics-controls" aria-label="Analytics view controls">
        <div className="analytics-switcher" role="group" aria-label="Analytics view">
          <button
            type="button"
            className={view === "portfolio" ? "active" : ""}
            onClick={() => setView("portfolio")}
          >
            Portfolio overview
          </button>
          <button
            type="button"
            className={view === "cid" ? "active" : ""}
            onClick={() => setView("cid")}
          >
            CID trends
          </button>
        </div>

        {view === "cid" && (
          <label className="cid-select">
            <span>Select CID</span>
            <select value={selectedCID} onChange={(event) => setSelectedCID(event.target.value)}>
              {trendFixture.cids.map((history) => (
                <option key={history.cid} value={history.cid}>
                  {compactTrendCID(history.cid)} / {history.label}
                </option>
              ))}
            </select>
          </label>
        )}
      </section>

      {view === "portfolio" ? <PortfolioDashboard /> : <CIDDashboard history={selectedHistory} />}
    </main>
  );
}

function PortfolioDashboard() {
  const summary = portfolioSummary();
  const trend = portfolioTrendSeries();
  const distribution = portfolioScoreDistribution();
  const zeroControls = zeroComplianceControls();
  const controls = recurringControlGaps();
  const platforms = platformComparison();
  const movers = portfolioCIDMovers();
  const scatter = portfolioScatterData();
  const strongestGain = movers[0];
  const largestDecline = movers.at(-1)!;
  const mostRecurring = controls[0];

  return (
    <div className="analytics-content">
      <section className="metric-grid" aria-label="Portfolio summary metrics">
        <MetricCard
          label="Median overall posture"
          value={`${score(summary.medianOverall)}/100`}
          detail={`${signed(summary.medianDelta)} points over 7 days`}
          direction={summary.medianDelta}
        />
        <MetricCard
          label="Organizations improving"
          value={`${summary.improvingCids} of ${summary.cidCount}`}
          detail="Equal-weight CID comparison"
          direction={summary.improvingCids - summary.cidCount / 2}
        />
        <MetricCard
          label="Reported endpoints"
          value={devices(summary.totalDevices)}
          detail="Latest synthetic snapshot"
        />
        <MetricCard
          label="Zero-control observations"
          value={String(summary.zeroGaps)}
          detail={`${signed(summary.zeroGapDelta)} net over 7 days`}
          direction={summary.zeroGapDelta == null ? null : -summary.zeroGapDelta}
        />
      </section>

      <section className="analytics-grid">
        <ChartPanel
          title="Portfolio posture direction"
          description="Equal-weight CID medians; the dashed line shows device-weighted overall posture."
          className="panel-wide"
        >
          <div className="chart-frame chart-frame-large">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={trend} margin={{ top: 12, right: 12, left: -14, bottom: 0 }}>
                <CartesianGrid stroke="#e3dfd7" strokeDasharray="3 5" vertical={false} />
                <XAxis dataKey="dateLabel" tickLine={false} axisLine={false} />
                <YAxis domain={[0, 100]} tickLine={false} axisLine={false} />
                <Tooltip content={<ChartTooltipCard />} />
                <Legend iconType="plainline" />
                <Line
                  type="monotone"
                  dataKey="overall"
                  name="Median overall"
                  stroke={chartColors.overall}
                  strokeWidth={3}
                  dot={{ r: 3 }}
                />
                <Line
                  type="monotone"
                  dataKey="os"
                  name="Median OS"
                  stroke={chartColors.os}
                  strokeWidth={2}
                  dot={false}
                  connectNulls
                />
                <Line
                  type="monotone"
                  dataKey="sensor"
                  name="Median sensor"
                  stroke={chartColors.sensor}
                  strokeWidth={2}
                  dot={false}
                  connectNulls
                />
                <Line
                  type="monotone"
                  dataKey="deviceWeighted"
                  name="Device-weighted overall"
                  stroke={chartColors.weighted}
                  strokeWidth={2}
                  strokeDasharray="6 5"
                  dot={false}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </ChartPanel>

        <ChartPanel
          title="Current score distribution"
          description="Latest overall scores grouped into fixed bands; no risk labels are inferred."
        >
          <div className="chart-frame">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={distribution} margin={{ top: 12, right: 8, left: -24, bottom: 12 }}>
                <CartesianGrid stroke="#e3dfd7" strokeDasharray="3 5" vertical={false} />
                <XAxis
                  dataKey="label"
                  tickLine={false}
                  axisLine={false}
                  tick={{ fontSize: 9 }}
                  interval={0}
                />
                <YAxis
                  domain={[0, summary.cidCount]}
                  allowDecimals={false}
                  tickLine={false}
                  axisLine={false}
                />
                <Tooltip
                  content={({ active, payload }) => {
                    const band = payload?.[0]?.payload as
                      | (typeof distribution)[number]
                      | undefined;
                    if (!active || !band) {
                      return null;
                    }
                    return (
                      <div className="chart-tooltip">
                        <strong>{band.label}</strong>
                        <span>CIDs: <b>{band.count}</b></span>
                        <span>{band.cids.length > 0 ? band.cids.join(", ") : "No CIDs"}</span>
                      </div>
                    );
                  }}
                />
                <Bar dataKey="count" name="CIDs" radius={[4, 4, 0, 0]}>
                  {distribution.map((band, index) => (
                    <Cell key={band.label} fill={scoreBandColors[index]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
          <p className="chart-footnote">
            Each CID contributes equally, regardless of endpoint volume.
          </p>
        </ChartPanel>

        <ChartPanel
          title="Zero-compliance controls"
          description="Controls reporting exactly zero in the largest number of applicable CIDs."
          className="panel-tall"
        >
          <div className="chart-frame chart-frame-tall">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={zeroControls}
                layout="vertical"
                margin={{ top: 5, right: 12, left: 20, bottom: 0 }}
              >
                <CartesianGrid stroke="#e3dfd7" strokeDasharray="3 5" horizontal={false} />
                <XAxis
                  type="number"
                  domain={[0, summary.cidCount]}
                  allowDecimals={false}
                  tickLine={false}
                />
                <YAxis
                  type="category"
                  dataKey="label"
                  width={150}
                  tick={{ fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                />
                <Tooltip
                  cursor={{ fill: "#f4eee8" }}
                  content={({ active, payload }) => {
                    const control = payload?.[0]?.payload as
                      | (typeof zeroControls)[number]
                      | undefined;
                    if (!active || !control) {
                      return null;
                    }
                    return (
                      <div className="chart-tooltip">
                        <strong>{control.label}</strong>
                        <span>{control.id}</span>
                        <span>
                          Zero in: <b>{control.zeroCids} of {control.applicableCids} CIDs</b>
                        </span>
                        <span>Zero platforms: <b>{control.platforms.join(", ")}</b></span>
                      </div>
                    );
                  }}
                />
                <Bar
                  dataKey="zeroCids"
                  name="Zero-compliance CIDs"
                  fill={chartColors.negative}
                  radius={[0, 4, 4, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          </div>
          <p className="chart-footnote">
            Denominators vary because controls are not applicable to every platform.
          </p>
        </ChartPanel>

        <ChartPanel
          title="Recurring control gaps"
          description="CIDs where a control is zero or among that platform's priority opportunities."
          className="panel-tall"
        >
          <div className="chart-frame chart-frame-tall">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={controls}
                layout="vertical"
                margin={{ top: 5, right: 12, left: 20, bottom: 0 }}
              >
                <CartesianGrid stroke="#e3dfd7" strokeDasharray="3 5" horizontal={false} />
                <XAxis type="number" domain={[0, trendFixture.cids.length]} allowDecimals={false} />
                <YAxis
                  type="category"
                  dataKey="label"
                  width={150}
                  tick={{ fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                />
                <Tooltip content={<ChartTooltipCard />} />
                <Bar
                  dataKey="affectedCids"
                  name="Affected CIDs"
                  fill={chartColors.weighted}
                  radius={[0, 4, 4, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          </div>
          <p className="chart-footnote">
            Counts are based on applicable CIDs, not the entire portfolio.
          </p>
        </ChartPanel>

        <ChartPanel
          title="OS and sensor balance"
          description="Each point is one CID; point size reflects reported endpoint volume."
        >
          <div className="chart-frame chart-frame-tall">
            <ResponsiveContainer width="100%" height="100%">
              <ScatterChart margin={{ top: 14, right: 18, bottom: 8, left: -8 }}>
                <CartesianGrid stroke="#e3dfd7" strokeDasharray="3 5" />
                <XAxis
                  type="number"
                  dataKey="os"
                  name="OS score"
                  domain={[0, 100]}
                  tickLine={false}
                  label={{ value: "OS score", position: "insideBottom", offset: -5 }}
                />
                <YAxis
                  type="number"
                  dataKey="sensor"
                  name="Sensor score"
                  domain={[0, 100]}
                  tickLine={false}
                  label={{ value: "Sensor", angle: -90, position: "insideLeft" }}
                />
                <ZAxis type="number" dataKey="devices" range={[75, 650]} />
                <Tooltip
                  cursor={{ strokeDasharray: "3 3" }}
                  content={({ active, payload }) => {
                    const datum = payload?.[0]?.payload as
                      | { label: string; compactCID: string; os: number; sensor: number; devices: number }
                      | undefined;
                    if (!active || !datum) {
                      return null;
                    }
                    return (
                      <div className="chart-tooltip scatter-tooltip">
                        <strong>{datum.label}</strong>
                        <span>{datum.compactCID}</span>
                        <span>OS: <b>{datum.os.toFixed(1)}</b></span>
                        <span>Sensor: <b>{datum.sensor.toFixed(1)}</b></span>
                        <span>Devices: <b>{datum.devices.toLocaleString()}</b></span>
                      </div>
                    );
                  }}
                />
                <Scatter data={scatter} fill={chartColors.overall} fillOpacity={0.78} />
              </ScatterChart>
            </ResponsiveContainer>
          </div>
        </ChartPanel>

        <ChartPanel
          title="Platform posture comparison"
          description="Median overall score on the first and final day among CIDs reporting each platform."
          className="panel-wide"
        >
          <div className="chart-frame">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={platforms} margin={{ top: 10, right: 10, left: -12, bottom: 25 }}>
                <CartesianGrid stroke="#e3dfd7" strokeDasharray="3 5" vertical={false} />
                <XAxis
                  dataKey="platform"
                  tickLine={false}
                  axisLine={false}
                  angle={-24}
                  textAnchor="end"
                  height={60}
                  interval={0}
                  tick={{ fontSize: 10 }}
                />
                <YAxis domain={[0, 100]} tickLine={false} axisLine={false} />
                <Tooltip content={<ChartTooltipCard />} />
                <Legend />
                <Bar dataKey="first" name="Day 1" fill={chartColors.baseline} radius={[3, 3, 0, 0]} />
                <Bar dataKey="current" name="Current" fill={chartColors.overall} radius={[3, 3, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </ChartPanel>

        <ChartPanel
          title="Weekly portfolio brief"
          description="Preview of the narrative a future scheduled Claude trend job could produce."
          className="brief-panel"
        >
          <div className="brief-label"><span /> Grounded demo preview</div>
          <h4>
            Portfolio posture moved {summary.medianDelta != null && summary.medianDelta >= 0 ? "up" : "down"},
            but recurring control gaps remain concentrated.
          </h4>
          <div className="brief-stat-grid">
            <div className="brief-stat">
              <span>Strongest gain</span>
              <strong>{strongestGain.label}</strong>
              <b>{signed(strongestGain.delta)} pts</b>
            </div>
            <div className="brief-stat">
              <span>Largest decline</span>
              <strong>{largestDecline.label}</strong>
              <b>{signed(largestDecline.delta)} pts</b>
            </div>
            <div className="brief-stat">
              <span>Persistent gap</span>
              <strong>{mostRecurring?.label ?? "No recurring gap"}</strong>
              <b>{mostRecurring ? `${mostRecurring.affectedCids} CIDs` : "None"}</b>
            </div>
          </div>
          <div className="brief-next-step">
            <span>Next-week focus</span>
            Validate the recurring controls against the affected platforms first, then review the two declining CIDs before broad remediation planning.
          </div>
        </ChartPanel>

        <ChartPanel
          title="CID movement"
          description="Seven-day overall score change, sorted from strongest improvement to largest decline."
          className="panel-full"
        >
          <div className="chart-frame chart-frame-tall">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={movers}
                layout="vertical"
                margin={{ top: 5, right: 28, left: 30, bottom: 0 }}
              >
                <CartesianGrid stroke="#e3dfd7" strokeDasharray="3 5" horizontal={false} />
                <XAxis
                  type="number"
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(value: number) => `${value > 0 ? "+" : ""}${value}`}
                />
                <YAxis
                  type="category"
                  dataKey="label"
                  width={150}
                  tick={{ fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                />
                <ReferenceLine x={0} stroke="#6d6f6d" />
                <Tooltip content={<ChartTooltipCard />} />
                <Bar dataKey="delta" name="Score change" radius={4}>
                  {movers.map((mover) => (
                    <Cell
                      key={mover.cid}
                      fill={mover.delta >= 0 ? chartColors.positive : chartColors.negative}
                    />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        </ChartPanel>
      </section>
    </div>
  );
}

function CIDDashboard({ history }: { history: CIDHistory }) {
  const summary = cidSummary(history);
  const trend = cidTrendSeries(history);
  const platformTrend = cidPlatformTrendSeries(history);
  const zeroTransitions = cidZeroTransitions(history);
  const transitionGroups: Record<ZeroTransitionStatus, ZeroControlTransition[]> = {
    resolved: zeroTransitions.filter((transition) => transition.status === "resolved"),
    introduced: zeroTransitions.filter((transition) => transition.status === "introduced"),
    unchanged: zeroTransitions.filter((transition) => transition.status === "unchanged"),
  };
  const transitionData = (
    Object.keys(transitionGroups) as ZeroTransitionStatus[]
  ).map((status) => ({
    status,
    name: transitionLabels[status],
    value: transitionGroups[status].length,
  }));
  const movers = cidControlMovers(history);
  const strongestImprovement = [...movers].filter((mover) => mover.delta > 0).sort((a, b) => b.delta - a.delta)[0];
  const largestDecline = [...movers].filter((mover) => mover.delta < 0).sort((a, b) => a.delta - b.delta)[0];
  const deviceDelta = summary.current.reportedDevices - trend[0].reportedDevices;

  return (
    <div className="analytics-content">
      <section className="cid-analytics-title">
        <div>
          <p className="eyebrow">{history.label}</p>
          <h3>{history.cid}</h3>
        </div>
        <span>{history.snapshots.at(-1)!.platforms.length} reporting platforms</span>
      </section>

      <section className="metric-grid" aria-label="CID trend summary metrics">
        <MetricCard
          label="Current overall posture"
          value={`${score(summary.current.overall)}/100`}
          detail={`${signed(summary.overallDelta)} points over 7 days`}
          direction={summary.overallDelta}
        />
        <MetricCard
          label="OS movement"
          value={score(summary.current.os)}
          detail={`${signed(summary.osDelta)} points`}
          direction={summary.osDelta}
        />
        <MetricCard
          label="Sensor movement"
          value={score(summary.current.sensor)}
          detail={`${signed(summary.sensorDelta)} points`}
          direction={summary.sensorDelta}
        />
        <MetricCard
          label="Current zero observations"
          value={String(summary.current.zeroGaps)}
          detail={`${signed(summary.zeroDelta)} net over 7 days`}
          direction={-summary.zeroDelta}
        />
      </section>

      <section className="analytics-grid">
        <ChartPanel
          title="CID posture direction"
          description="Daily score movement for the selected organization. Missing components remain unavailable rather than displaying as zero."
          className="panel-wide"
        >
          <div className="chart-frame chart-frame-large">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={trend} margin={{ top: 12, right: 12, left: -14, bottom: 0 }}>
                <CartesianGrid stroke="#e3dfd7" strokeDasharray="3 5" vertical={false} />
                <XAxis dataKey="dateLabel" tickLine={false} axisLine={false} />
                <YAxis domain={[0, 100]} tickLine={false} axisLine={false} />
                <Tooltip content={<ChartTooltipCard />} />
                <Legend iconType="plainline" />
                <Line type="monotone" dataKey="overall" name="Overall" stroke={chartColors.overall} strokeWidth={3} dot={{ r: 3 }} />
                <Line type="monotone" dataKey="os" name="OS" stroke={chartColors.os} strokeWidth={2} dot={false} connectNulls />
                <Line type="monotone" dataKey="sensor" name="Sensor" stroke={chartColors.sensor} strokeWidth={2} dot={false} connectNulls />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </ChartPanel>

        <ChartPanel
          title="Zero-control transitions"
          description="Status of controls that were zero on day one or are zero in the current snapshot."
        >
          <div className="transition-visual">
            <div className="transition-donut">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={transitionData}
                    dataKey="value"
                    nameKey="name"
                    innerRadius="62%"
                    outerRadius="88%"
                    paddingAngle={3}
                    stroke="#fffdf9"
                    strokeWidth={3}
                  >
                    {transitionData.map((transition) => (
                      <Cell
                        key={transition.status}
                        fill={transitionColors[transition.status]}
                      />
                    ))}
                  </Pie>
                  <Tooltip content={<ChartTooltipCard />} />
                </PieChart>
              </ResponsiveContainer>
              <div className="donut-center">
                <strong>{zeroTransitions.length}</strong>
                <span>tracked states</span>
              </div>
            </div>
            <div className="transition-grid">
              {transitionData.map((transition) => (
                <div className="transition-card" key={transition.status}>
                  <span>
                    <i style={{ background: transitionColors[transition.status] }} />
                    {transition.name}
                  </span>
                  <strong>{transition.value}</strong>
                  <small>{transitionPreview(transitionGroups[transition.status])}</small>
                </div>
              ))}
            </div>
          </div>
        </ChartPanel>

        <ChartPanel
          title="Platform posture direction"
          description="Daily overall posture for every platform represented by this CID."
          className="panel-wide"
        >
          <div className="chart-frame chart-frame-large">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart
                data={platformTrend.points}
                margin={{ top: 12, right: 12, left: -14, bottom: 0 }}
              >
                <CartesianGrid stroke="#e3dfd7" strokeDasharray="3 5" vertical={false} />
                <XAxis dataKey="dateLabel" tickLine={false} axisLine={false} />
                <YAxis domain={[0, 100]} tickLine={false} axisLine={false} />
                <Tooltip content={<ChartTooltipCard />} />
                <Legend iconType="plainline" />
                {platformTrend.platforms.map((platform, index) => (
                  <Line
                    key={platform.key}
                    type="monotone"
                    dataKey={platform.key}
                    name={platform.label}
                    stroke={platformColors[index % platformColors.length]}
                    strokeWidth={index === 0 ? 3 : 2}
                    dot={index === 0 ? { r: 3 } : false}
                    connectNulls
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </div>
        </ChartPanel>

        <ChartPanel
          title="Assessed device volume"
          description={`${devices(trend[0].reportedDevices)} to ${devices(summary.current.reportedDevices)} devices over the demo week.`}
        >
          <div className="chart-frame">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={trend} margin={{ top: 14, right: 12, left: -20, bottom: 0 }}>
                <defs>
                  <linearGradient id="deviceVolumeFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor={chartColors.weighted} stopOpacity={0.34} />
                    <stop offset="95%" stopColor={chartColors.weighted} stopOpacity={0.03} />
                  </linearGradient>
                </defs>
                <CartesianGrid stroke="#e3dfd7" strokeDasharray="3 5" vertical={false} />
                <XAxis dataKey="dateLabel" tickLine={false} axisLine={false} />
                <YAxis allowDecimals={false} tickLine={false} axisLine={false} />
                <Tooltip content={<ChartTooltipCard />} />
                <Area
                  type="monotone"
                  dataKey="reportedDevices"
                  name="Reported devices"
                  stroke={chartColors.weighted}
                  strokeWidth={3}
                  fill="url(#deviceVolumeFill)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
          <p className="chart-footnote">
            Population change: {signed(deviceDelta)} devices. Score movement should be read in this context.
          </p>
        </ChartPanel>

        <ChartPanel
          title="Control movement"
          description="Largest absolute compliance changes among zero, priority, and recurring controls."
          className="panel-wide"
        >
          <div className="analytics-table-scroll">
            <table className="analytics-table">
              <thead>
                <tr>
                  <th>Control</th>
                  <th>Platform</th>
                  <th>Day 1</th>
                  <th>Current</th>
                  <th>Change</th>
                </tr>
              </thead>
              <tbody>
                {movers.slice(0, 8).map((mover) => (
                  <tr key={mover.key}>
                    <td>
                      <strong>{mover.label}</strong>
                      <code>{mover.id}</code>
                    </td>
                    <td>{mover.platform}</td>
                    <td>{(mover.start * 100).toFixed(1)}%</td>
                    <td>{(mover.current * 100).toFixed(1)}%</td>
                    <td>
                      <span className={`delta-pill ${directionClass(mover.delta)}`}>
                        {signed(mover.delta, " pp")}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </ChartPanel>

        <ChartPanel
          title="Weekly CID brief"
          description="Preview of a future Claude interpretation generated from calculated facts, not raw arithmetic."
          className="brief-panel"
        >
          <div className="brief-label"><span /> Grounded demo preview</div>
          <h4>
            Overall posture {summary.overallDelta != null && summary.overallDelta >= 0 ? "improved" : "declined"} by {signed(summary.overallDelta)} points during the seven-day period.
          </h4>
          <div className="brief-stat-grid">
            <div className="brief-stat">
              <span>Strongest control</span>
              <strong>{strongestImprovement?.label ?? "No improvement"}</strong>
              <b>{strongestImprovement ? signed(strongestImprovement.delta, " pp") : "None"}</b>
            </div>
            <div className="brief-stat">
              <span>Largest decline</span>
              <strong>{largestDecline?.label ?? "No decline"}</strong>
              <b>{largestDecline ? signed(largestDecline.delta, " pp") : "None"}</b>
            </div>
            <div className="brief-stat">
              <span>Zero state</span>
              <strong>{transitionGroups.resolved.length} resolved</strong>
              <b>{transitionGroups.introduced.length} introduced</b>
            </div>
          </div>
          <div className="brief-next-step">
            <span>Analyst focus</span>
            Review declining and newly zero controls first, then use the existing assessment narrative for platform-specific remediation and verification guidance.
          </div>
        </ChartPanel>
      </section>
    </div>
  );
}
