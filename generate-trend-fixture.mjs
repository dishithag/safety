import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const demoUIRoot = path.resolve(scriptDirectory, "..");
const sourceReportsDirectory = process.env.SOURCE_REPORTS_DIR
  ? path.resolve(process.env.SOURCE_REPORTS_DIR)
  : path.resolve(demoUIRoot, "..", "testdata", "sample_audit_reports", "cids");
const outputDirectory = path.resolve(demoUIRoot, "src", "demo-data", "trends");
const seedOutputDirectory = path.join(outputDirectory, "seed-reports");

const selectedReports = [
  {
    cid: "00000000000000000000000000000009c",
    label: "Mixed endpoint pilot",
    scoreDeltas: { overall: 5.3, os: 4.8, sensor: 5.9 },
  },
  {
    cid: "0000000000000000000000000000007b2",
    label: "Windows modernization",
    scoreDeltas: { overall: 2.4, os: 3.1, sensor: 1.8 },
  },
  {
    cid: "22cd0000000000000000000000000004c",
    label: "Foundational recovery",
    scoreDeltas: { overall: 1.6, os: 1.4, sensor: 1.9 },
  },
  {
    cid: "11ab00000000000000000000000000018",
    label: "Mature Windows estate",
    scoreDeltas: { overall: 0.8, os: 1.1, sensor: 0.5 },
  },
  {
    cid: "0f53593ceae34995af8fd295c18f1e25",
    label: "Enterprise workstation fleet",
    scoreDeltas: { overall: -1.2, os: -1.6, sensor: -0.7 },
  },
  {
    cid: "3a3a3a3a000044448888cccc12345678",
    label: "Server and Linux estate",
    scoreDeltas: { overall: 1.7, os: 1.3, sensor: 2.1 },
  },
  {
    cid: "9e8d7c6b5a4039281706f5e4d3c2b1a0",
    label: "Mobile and macOS fleet",
    scoreDeltas: { overall: -0.6, os: -0.9, sensor: -0.3 },
  },
  {
    cid: "deadbeefcafef00d0123456789abcdef",
    label: "Large multi-platform estate",
    scoreDeltas: { overall: 2.2, os: 1.9, sensor: 2.5 },
  },
];

const dates = [
  "2026-07-15",
  "2026-07-16",
  "2026-07-17",
  "2026-07-18",
  "2026-07-19",
  "2026-07-20",
  "2026-07-21",
];
const progress = [0, 0.14, 0.31, 0.49, 0.68, 0.84, 1];

function hashString(value) {
  let hash = 2166136261;
  for (const character of value) {
    hash ^= character.charCodeAt(0);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}

function round(value, precision = 2) {
  const factor = 10 ** precision;
  return Math.round((value + Number.EPSILON) * factor) / factor;
}

function clamp(value, minimum, maximum) {
  return Math.min(maximum, Math.max(minimum, value));
}

function pointSeries(finalValue, delta, key, precision = 2, maximum = 100) {
  if (finalValue == null) {
    return dates.map(() => null);
  }

  const start = clamp(finalValue - delta, 0, maximum);
  const hash = hashString(key);
  return progress.map((position, index) => {
    if (index === progress.length - 1) {
      return finalValue;
    }
    const jitterDirection = ((hash >> (index % 12)) & 1) === 0 ? -1 : 1;
    const jitter = index === 0 ? 0 : jitterDirection * ((hash % 7) / 35);
    return round(clamp(start + (finalValue - start) * position + jitter, 0, maximum), precision);
  });
}

function deviceSeries(finalValue, key) {
  const hash = hashString(key);
  const change = Math.max(1, Math.round(finalValue * (0.004 + (hash % 13) / 1000)));
  const start = Math.max(1, finalValue - change);
  return progress.map((position, index) =>
    index === progress.length - 1 ? finalValue : Math.round(start + (finalValue - start) * position),
  );
}

function signalSeries(finalValue, key) {
  const hash = hashString(key);

  if (finalValue === 0) {
    if (hash % 4 === 0) {
      const start = 0.06 + (hash % 9) / 100;
      return progress.map((position, index) =>
        index === progress.length - 1 ? 0 : round(clamp(start * (1 - position), 0, 1), 4),
      );
    }
    return dates.map(() => 0);
  }

  if (finalValue <= 0.25 && hash % 5 === 0) {
    return progress.map((position, index) => {
      if (index === progress.length - 1) {
        return finalValue;
      }
      return round(clamp(finalValue * Math.max(0, position - 0.12) / 0.88, 0, 1), 4);
    });
  }

  const signedChange = ((hash % 27) - 10) / 100;
  return pointSeries(finalValue, signedChange, key, 4, 1);
}

function prioritySignalIDs(compliance) {
  const partial = Object.entries(compliance)
    .filter(([, value]) => value > 0 && value < 0.9999)
    .sort((left, right) => left[1] - right[1] || left[0].localeCompare(right[0]));
  if (partial.length === 0) {
    return new Set();
  }
  const cutoff = partial[Math.min(2, partial.length - 1)][1];
  return new Set(partial.filter(([, value]) => value <= cutoff).map(([id]) => id));
}

function signalOccurrenceStats(reports) {
  const stats = new Map();
  for (const report of reports) {
    for (const platform of report.platforms) {
      for (const [id, compliance] of Object.entries(platform.compliance)) {
        const current = stats.get(id) ?? { id, count: 0, sum: 0, zeroCount: 0 };
        current.count += 1;
        current.sum += compliance;
        if (compliance === 0) {
          current.zeroCount += 1;
        }
        stats.set(id, current);
      }
    }
  }
  return [...stats.values()]
    .filter((signal) => signal.count >= 2)
    .sort(
      (left, right) =>
        right.zeroCount - left.zeroCount ||
        left.sum / left.count - right.sum / right.count ||
        right.count - left.count,
    )
    .slice(0, 16)
    .map((signal) => signal.id);
}

function selectedSignals(platform, recurringSignalIDs) {
  const priorities = prioritySignalIDs(platform.compliance);
  const entries = Object.entries(platform.compliance);
  const bottomPartial = entries
    .filter(([, value]) => value > 0 && value < 0.9999)
    .sort((left, right) => left[1] - right[1] || left[0].localeCompare(right[0]))
    .slice(0, 6)
    .map(([id]) => id);
  const selectedIDs = new Set([
    ...entries.filter(([, value]) => value === 0).map(([id]) => id),
    ...bottomPartial,
    ...entries.filter(([id]) => recurringSignalIDs.has(id)).map(([id]) => id),
  ]);

  return entries
    .filter(([id]) => selectedIDs.has(id))
    .sort((left, right) => left[1] - right[1] || left[0].localeCompare(right[0]))
    .map(([id, compliance]) => ({
      id,
      compliance,
      focus: compliance === 0 ? "zero" : priorities.has(id) ? "priority" : "tracked",
    }));
}

function buildCIDHistory(selection, report, recurringSignalIDs) {
  const overallSeries = pointSeries(
    report.average_overall_score,
    selection.scoreDeltas.overall,
    `${report.cid}:overall`,
  );
  const osSeries = pointSeries(
    report.average_os_score,
    selection.scoreDeltas.os,
    `${report.cid}:os`,
  );
  const sensorSeries = pointSeries(
    report.average_sensor_config_score,
    selection.scoreDeltas.sensor,
    `${report.cid}:sensor`,
  );
  const reportedDevices = deviceSeries(report.num_aids, `${report.cid}:devices`);

  const platformSeries = report.platforms.map((platform) => {
    const variation = ((hashString(`${report.cid}:${platform.name}`) % 13) - 6) / 10;
    const platformDelta = selection.scoreDeltas.overall + variation;
    const signals = selectedSignals(platform, recurringSignalIDs).map((signal) => ({
      ...signal,
      series: signalSeries(signal.compliance, `${report.cid}:${platform.name}:${signal.id}`),
    }));
    return {
      source: platform,
      devices: deviceSeries(platform.num_aids, `${report.cid}:${platform.name}:devices`),
      overall: pointSeries(
        platform.average_overall_score,
        platformDelta,
        `${report.cid}:${platform.name}:overall`,
      ),
      os: pointSeries(
        platform.average_os_score,
        selection.scoreDeltas.os + variation,
        `${report.cid}:${platform.name}:os`,
      ),
      sensor: pointSeries(
        platform.average_sensor_config_score,
        selection.scoreDeltas.sensor + variation,
        `${report.cid}:${platform.name}:sensor`,
      ),
      signals,
    };
  });

  return {
    cid: report.cid,
    label: selection.label,
    snapshots: dates.map((date, index) => ({
      date,
      reported_devices: reportedDevices[index],
      scores: {
        overall: overallSeries[index],
        os: osSeries[index],
        sensor: sensorSeries[index],
      },
      platforms: platformSeries.map((platform) => ({
        name: platform.source.name,
        reported_devices: platform.devices[index],
        scores: {
          overall: platform.overall[index],
          os: platform.os[index],
          sensor: platform.sensor[index],
        },
        signals: platform.signals.map((signal) => ({
          id: signal.id,
          compliance: signal.series[index],
          focus: signal.focus,
        })),
      })),
    })),
  };
}

await mkdir(seedOutputDirectory, { recursive: true });

const reports = [];
for (const selection of selectedReports) {
  const sourcePath = path.join(sourceReportsDirectory, `${selection.cid}.json`);
  const report = JSON.parse(await readFile(sourcePath, "utf8"));
  if (report.cid !== selection.cid) {
    throw new Error(`CID mismatch in ${sourcePath}: got ${report.cid}`);
  }
  reports.push(report);
  await writeFile(
    path.join(seedOutputDirectory, `${selection.cid}.json`),
    `${JSON.stringify(
      {
        demo_notice: "Copied from synthetic repository fixtures for the isolated trend demo.",
        report,
      },
      null,
      2,
    )}\n`,
  );
}

const recurringSignalIDs = new Set(signalOccurrenceStats(reports));
const fixture = {
  metadata: {
    synthetic: true,
    purpose: "Future-scope weekly trend analytics demonstration",
    generated_from: "Synthetic repository CID fixtures",
    period_start: dates[0],
    period_end: dates[dates.length - 1],
    cadence: "daily",
    snapshot_count: dates.length,
    cid_count: selectedReports.length,
  },
  cids: selectedReports.map((selection) => {
    const report = reports.find((candidate) => candidate.cid === selection.cid);
    return buildCIDHistory(selection, report, recurringSignalIDs);
  }),
};

await writeFile(
  path.join(outputDirectory, "portfolio-week.json"),
  `${JSON.stringify(fixture, null, 2)}\n`,
);

console.log(
  `Generated ${fixture.cids.length} CID histories with ${dates.length} snapshots each in ${outputDirectory}`,
);

