import fixtureJSON from "../demo-data/trends/portfolio-week.json";

export type ScoreSet = {
  overall: number | null;
  os: number | null;
  sensor: number | null;
};

export type SignalFocus = "zero" | "priority" | "tracked";

export type SignalSnapshot = {
  id: string;
  compliance: number;
  focus: SignalFocus;
};

export type PlatformSnapshot = {
  name: string;
  reported_devices: number;
  scores: ScoreSet;
  signals: SignalSnapshot[];
};

export type DailySnapshot = {
  date: string;
  reported_devices: number;
  scores: ScoreSet;
  platforms: PlatformSnapshot[];
};

export type CIDHistory = {
  cid: string;
  label: string;
  snapshots: DailySnapshot[];
};

export type TrendFixture = {
  metadata: {
    synthetic: boolean;
    purpose: string;
    generated_from: string;
    period_start: string;
    period_end: string;
    cadence: string;
    snapshot_count: number;
    cid_count: number;
  };
  cids: CIDHistory[];
};

export const trendFixture = fixtureJSON as TrendFixture;

export type PortfolioTrendPoint = {
  date: string;
  dateLabel: string;
  overall: number | null;
  os: number | null;
  sensor: number | null;
  deviceWeighted: number | null;
  zeroGaps: number;
};

export type RecurringControlGap = {
  id: string;
  label: string;
  affectedCids: number;
  applicableCids: number;
  zeroCids: number;
  medianCompliance: number;
};

export type ControlMover = {
  key: string;
  id: string;
  label: string;
  platform: string;
  start: number;
  current: number;
  delta: number;
  focus: SignalFocus;
};

export type CIDMover = {
  cid: string;
  label: string;
  compactCID: string;
  start: number;
  current: number;
  delta: number;
};

export type ScoreDistributionBand = {
  label: string;
  count: number;
  cids: string[];
};

export type ZeroComplianceControl = {
  id: string;
  label: string;
  zeroCids: number;
  applicableCids: number;
  platforms: string[];
};

export type PlatformTrendPoint = {
  date: string;
  dateLabel: string;
  [key: string]: string | number | null;
};

export type ZeroTransitionStatus = "resolved" | "introduced" | "unchanged";

export type ZeroControlTransition = {
  key: string;
  id: string;
  label: string;
  platform: string;
  status: ZeroTransitionStatus;
};

function finiteValues(values: Array<number | null>): number[] {
  return values.filter((value): value is number => value != null && Number.isFinite(value));
}

export function median(values: Array<number | null>): number | null {
  const sorted = finiteValues(values).sort((left, right) => left - right);
  if (sorted.length === 0) {
    return null;
  }
  const middle = Math.floor(sorted.length / 2);
  if (sorted.length % 2 === 1) {
    return sorted[middle];
  }
  return (sorted[middle - 1] + sorted[middle]) / 2;
}

export function roundMetric(value: number | null, precision = 2): number | null {
  if (value == null) {
    return null;
  }
  const factor = 10 ** precision;
  return Math.round((value + Number.EPSILON) * factor) / factor;
}

export function dateLabel(date: string): string {
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  }).format(new Date(`${date}T00:00:00Z`));
}

export function compactTrendCID(cid: string): string {
  return `${cid.slice(0, 5)}...${cid.slice(-4)}`;
}

const signalLabelOverrides: Record<string, string> = {
  credential_guard_running: "Credential Guard Running",
  dma_guard_enabled: "DMA Guard Enabled",
  hvci_enabled: "HVCI Enabled",
  hvci_strict_mode: "HVCI Strict Mode",
  iommu_in_use: "IOMMU In Use",
  real_time_response: "Real Time Response",
  secure_boot_enabled: "Secure Boot Enabled",
  secure_kernel_running: "Secure Kernel Running",
  smm_protections: "SMM Protections",
  uefi_memory_protection: "UEFI Memory Protection",
  vsm_available: "VSM Available",
};

export function signalLabel(id: string): string {
  if (signalLabelOverrides[id]) {
    return signalLabelOverrides[id];
  }
  return id
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

function firstSnapshot(history: CIDHistory): DailySnapshot {
  return history.snapshots[0];
}

function latestSnapshot(history: CIDHistory): DailySnapshot {
  return history.snapshots[history.snapshots.length - 1];
}

function countZeroSignals(snapshot: DailySnapshot): number {
  return snapshot.platforms.reduce(
    (total, platform) =>
      total + platform.signals.filter((signal) => signal.compliance === 0).length,
    0,
  );
}

export function portfolioTrendSeries(): PortfolioTrendPoint[] {
  return trendFixture.cids[0].snapshots.map((snapshot, index) => {
    const snapshots = trendFixture.cids.map((history) => history.snapshots[index]);
    const weightedValues = snapshots.filter((current) => current.scores.overall != null);
    const totalDevices = weightedValues.reduce((total, current) => total + current.reported_devices, 0);
    const weightedOverall =
      totalDevices === 0
        ? null
        : weightedValues.reduce(
            (total, current) => total + (current.scores.overall ?? 0) * current.reported_devices,
            0,
          ) / totalDevices;

    return {
      date: snapshot.date,
      dateLabel: dateLabel(snapshot.date),
      overall: roundMetric(median(snapshots.map((current) => current.scores.overall))),
      os: roundMetric(median(snapshots.map((current) => current.scores.os))),
      sensor: roundMetric(median(snapshots.map((current) => current.scores.sensor))),
      deviceWeighted: roundMetric(weightedOverall),
      zeroGaps: snapshots.reduce((total, current) => total + countZeroSignals(current), 0),
    };
  });
}

export function portfolioSummary() {
  const first = portfolioTrendSeries()[0];
  const current = portfolioTrendSeries().at(-1)!;
  const latestSnapshots = trendFixture.cids.map(latestSnapshot);
  const totalDevices = latestSnapshots.reduce((total, snapshot) => total + snapshot.reported_devices, 0);
  const improvingCids = trendFixture.cids.filter((history) => {
    const start = firstSnapshot(history).scores.overall;
    const latest = latestSnapshot(history).scores.overall;
    return start != null && latest != null && latest > start;
  }).length;

  let resolvedZeros = 0;
  let introducedZeros = 0;
  let unchangedZeros = 0;
  for (const history of trendFixture.cids) {
    const start = firstSnapshot(history);
    const latest = latestSnapshot(history);
    for (const latestPlatform of latest.platforms) {
      const startPlatform = start.platforms.find((platform) => platform.name === latestPlatform.name);
      for (const latestSignal of latestPlatform.signals) {
        const startSignal = startPlatform?.signals.find((signal) => signal.id === latestSignal.id);
        if (!startSignal) {
          continue;
        }
        if (startSignal.compliance === 0 && latestSignal.compliance > 0) {
          resolvedZeros += 1;
        }
        if (startSignal.compliance > 0 && latestSignal.compliance === 0) {
          introducedZeros += 1;
        }
        if (startSignal.compliance === 0 && latestSignal.compliance === 0) {
          unchangedZeros += 1;
        }
      }
    }
  }

  return {
    cidCount: trendFixture.cids.length,
    totalDevices,
    medianOverall: current.overall,
    medianDelta:
      current.overall == null || first.overall == null
        ? null
        : roundMetric(current.overall - first.overall),
    zeroGaps: current.zeroGaps,
    zeroGapDelta: current.zeroGaps - first.zeroGaps,
    resolvedZeros,
    introducedZeros,
    unchangedZeros,
    improvingCids,
  };
}

export function portfolioScoreDistribution(): ScoreDistributionBand[] {
  const bands: ScoreDistributionBand[] = [
    { label: "0 to <20", count: 0, cids: [] },
    { label: "20 to <40", count: 0, cids: [] },
    { label: "40 to <60", count: 0, cids: [] },
    { label: "60 to <80", count: 0, cids: [] },
    { label: "80 to 100", count: 0, cids: [] },
  ];

  for (const history of trendFixture.cids) {
    const overall = latestSnapshot(history).scores.overall;
    if (overall == null || !Number.isFinite(overall)) {
      continue;
    }
    const bandIndex = Math.min(4, Math.max(0, Math.floor(overall / 20)));
    bands[bandIndex].count += 1;
    bands[bandIndex].cids.push(compactTrendCID(history.cid));
  }

  return bands;
}

export function portfolioScatterData() {
  return trendFixture.cids.flatMap((history) => {
    const current = latestSnapshot(history);
    if (current.scores.os == null || current.scores.sensor == null) {
      return [];
    }
    return [
      {
        cid: history.cid,
        compactCID: compactTrendCID(history.cid),
        label: history.label,
        os: current.scores.os,
        sensor: current.scores.sensor,
        overall: current.scores.overall,
        devices: current.reported_devices,
      },
    ];
  });
}

export function zeroComplianceControls(limit = 8): ZeroComplianceControl[] {
  const controls = new Map<
    string,
    {
      applicableCids: Set<string>;
      zeroCids: Set<string>;
      platforms: Set<string>;
    }
  >();

  for (const history of trendFixture.cids) {
    for (const platform of latestSnapshot(history).platforms) {
      for (const signal of platform.signals) {
        const current = controls.get(signal.id) ?? {
          applicableCids: new Set<string>(),
          zeroCids: new Set<string>(),
          platforms: new Set<string>(),
        };
        current.applicableCids.add(history.cid);
        if (signal.compliance === 0) {
          current.zeroCids.add(history.cid);
          current.platforms.add(platform.name);
        }
        controls.set(signal.id, current);
      }
    }
  }

  return [...controls.entries()]
    .map(([id, values]) => ({
      id,
      label: signalLabel(id),
      zeroCids: values.zeroCids.size,
      applicableCids: values.applicableCids.size,
      platforms: [...values.platforms].sort(),
    }))
    .filter((control) => control.zeroCids > 0)
    .sort(
      (left, right) =>
        right.zeroCids - left.zeroCids ||
        right.zeroCids / right.applicableCids - left.zeroCids / left.applicableCids ||
        left.label.localeCompare(right.label),
    )
    .slice(0, limit);
}

export function recurringControlGaps(limit = 8): RecurringControlGap[] {
  const controls = new Map<
    string,
    {
      applicableCids: Set<string>;
      affectedCids: Set<string>;
      zeroCids: Set<string>;
      compliance: number[];
    }
  >();

  for (const history of trendFixture.cids) {
    for (const platform of latestSnapshot(history).platforms) {
      for (const signal of platform.signals) {
        const current = controls.get(signal.id) ?? {
          applicableCids: new Set<string>(),
          affectedCids: new Set<string>(),
          zeroCids: new Set<string>(),
          compliance: [],
        };
        current.applicableCids.add(history.cid);
        current.compliance.push(signal.compliance);
        if (signal.focus === "zero" || signal.focus === "priority") {
          current.affectedCids.add(history.cid);
        }
        if (signal.compliance === 0) {
          current.zeroCids.add(history.cid);
        }
        controls.set(signal.id, current);
      }
    }
  }

  return [...controls.entries()]
    .map(([id, values]) => ({
      id,
      label: signalLabel(id),
      affectedCids: values.affectedCids.size,
      applicableCids: values.applicableCids.size,
      zeroCids: values.zeroCids.size,
      medianCompliance: median(values.compliance) ?? 0,
    }))
    .filter((control) => control.affectedCids > 0)
    .sort(
      (left, right) =>
        right.affectedCids - left.affectedCids ||
        right.zeroCids - left.zeroCids ||
        left.medianCompliance - right.medianCompliance,
    )
    .slice(0, limit);
}

export function platformComparison() {
  const platforms = new Map<string, { first: number[]; current: number[]; cids: Set<string> }>();
  for (const history of trendFixture.cids) {
    const start = firstSnapshot(history);
    const latest = latestSnapshot(history);
    for (const platform of latest.platforms) {
      const current = platforms.get(platform.name) ?? {
        first: [],
        current: [],
        cids: new Set<string>(),
      };
      const startPlatform = start.platforms.find((candidate) => candidate.name === platform.name);
      if (startPlatform?.scores.overall != null) {
        current.first.push(startPlatform.scores.overall);
      }
      if (platform.scores.overall != null) {
        current.current.push(platform.scores.overall);
      }
      current.cids.add(history.cid);
      platforms.set(platform.name, current);
    }
  }

  return [...platforms.entries()]
    .map(([platform, values]) => ({
      platform,
      first: roundMetric(median(values.first)),
      current: roundMetric(median(values.current)),
      cids: values.cids.size,
    }))
    .sort((left, right) => (right.current ?? 0) - (left.current ?? 0));
}

export function portfolioCIDMovers(): CIDMover[] {
  return trendFixture.cids
    .map((history) => {
      const start = firstSnapshot(history).scores.overall ?? 0;
      const current = latestSnapshot(history).scores.overall ?? 0;
      return {
        cid: history.cid,
        label: history.label,
        compactCID: compactTrendCID(history.cid),
        start,
        current,
        delta: roundMetric(current - start) ?? 0,
      };
    })
    .sort((left, right) => right.delta - left.delta);
}

export function cidTrendSeries(history: CIDHistory) {
  return history.snapshots.map((snapshot) => ({
    date: snapshot.date,
    dateLabel: dateLabel(snapshot.date),
    overall: snapshot.scores.overall,
    os: snapshot.scores.os,
    sensor: snapshot.scores.sensor,
    zeroGaps: countZeroSignals(snapshot),
    reportedDevices: snapshot.reported_devices,
  }));
}

export function cidPlatformTrendSeries(history: CIDHistory) {
  const platformNames = [
    ...new Set(
      history.snapshots.flatMap((snapshot) =>
        snapshot.platforms.map((platform) => platform.name),
      ),
    ),
  ];
  const platforms = platformNames.map((label, index) => ({
    key: `platform_${index}`,
    label,
  }));
  const points: PlatformTrendPoint[] = history.snapshots.map((snapshot) => {
    const point: PlatformTrendPoint = {
      date: snapshot.date,
      dateLabel: dateLabel(snapshot.date),
    };
    for (const platform of platforms) {
      point[platform.key] =
        snapshot.platforms.find((candidate) => candidate.name === platform.label)?.scores
          .overall ?? null;
    }
    return point;
  });

  return { platforms, points };
}

export function cidZeroTransitions(history: CIDHistory): ZeroControlTransition[] {
  const start = firstSnapshot(history);
  const current = latestSnapshot(history);
  const startSignals = new Map<string, SignalSnapshot>();

  for (const platform of start.platforms) {
    for (const signal of platform.signals) {
      startSignals.set(`${platform.name}:${signal.id}`, signal);
    }
  }

  const transitions: ZeroControlTransition[] = [];
  for (const platform of current.platforms) {
    for (const signal of platform.signals) {
      const key = `${platform.name}:${signal.id}`;
      const startSignal = startSignals.get(key);
      if (!startSignal) {
        continue;
      }

      let status: ZeroTransitionStatus | null = null;
      if (startSignal.compliance === 0 && signal.compliance > 0) {
        status = "resolved";
      } else if (startSignal.compliance > 0 && signal.compliance === 0) {
        status = "introduced";
      } else if (startSignal.compliance === 0 && signal.compliance === 0) {
        status = "unchanged";
      }
      if (status === null) {
        continue;
      }

      transitions.push({
        key,
        id: signal.id,
        label: signalLabel(signal.id),
        platform: platform.name,
        status,
      });
    }
  }

  return transitions.sort(
    (left, right) =>
      left.status.localeCompare(right.status) ||
      left.platform.localeCompare(right.platform) ||
      left.label.localeCompare(right.label),
  );
}

export function cidControlMovers(history: CIDHistory): ControlMover[] {
  const start = firstSnapshot(history);
  const current = latestSnapshot(history);
  const movers: ControlMover[] = [];

  for (const platform of current.platforms) {
    const startPlatform = start.platforms.find((candidate) => candidate.name === platform.name);
    for (const signal of platform.signals) {
      const startSignal = startPlatform?.signals.find((candidate) => candidate.id === signal.id);
      if (!startSignal) {
        continue;
      }
      movers.push({
        key: `${platform.name}:${signal.id}`,
        id: signal.id,
        label: signalLabel(signal.id),
        platform: platform.name,
        start: startSignal.compliance,
        current: signal.compliance,
        delta: roundMetric((signal.compliance - startSignal.compliance) * 100) ?? 0,
        focus: signal.focus,
      });
    }
  }

  return movers.sort((left, right) => Math.abs(right.delta) - Math.abs(left.delta));
}

export function cidSummary(history: CIDHistory) {
  const series = cidTrendSeries(history);
  const start = series[0];
  const current = series.at(-1)!;
  const movers = cidControlMovers(history);
  return {
    current,
    overallDelta:
      current.overall == null || start.overall == null
        ? null
        : roundMetric(current.overall - start.overall),
    osDelta:
      current.os == null || start.os == null ? null : roundMetric(current.os - start.os),
    sensorDelta:
      current.sensor == null || start.sensor == null
        ? null
        : roundMetric(current.sensor - start.sensor),
    zeroDelta: current.zeroGaps - start.zeroGaps,
    strongestMover: movers[0] ?? null,
    improvingControls: movers.filter((mover) => mover.delta > 0).length,
    decliningControls: movers.filter((mover) => mover.delta < 0).length,
  };
}
