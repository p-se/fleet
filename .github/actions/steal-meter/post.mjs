import * as fs from "fs";
import * as core from "@actions/core";

function readProcStat() {
  const ln = fs
    .readFileSync("/proc/stat", "utf8")
    .split("\n")
    .find((l) => l.startsWith("cpu "));
  const parts = ln.trim().split(/\s+/).slice(1).map(Number);
  const [user, nice, system, idle, iowait, irq, softirq, steal] = parts;
  const total = user + nice + system + idle + iowait + irq + softirq + steal;
  return { total, steal };
}

try {
  const startTotal = Number(core.getState("start_total"));
  const startSteal = Number(core.getState("start_steal"));
  const { total: endTotal, steal: endSteal } = readProcStat();

  const dTotal = endTotal - startTotal;
  const dSteal = endSteal - startSteal;

  if (dTotal > 0) {
    const pct = ((100 * dSteal) / dTotal).toFixed(2);
    core.summary
      .addHeading("Runner Steal Time")
      .addTable([
        ["Metric", "Value"],
        ["Average %st over job", `${pct}%`],
        ["Δsteal jiffies", String(dSteal)],
        ["Δtotal jiffies", String(dTotal)],
      ])
      .write();
    core.info(
      `Average steal across job: ${pct}% (Δsteal=${dSteal}, Δtotal=${dTotal})`,
    );
  } else {
    core.info("No CPU ticks advanced; cannot compute steal.");
  }
} catch (e) {
  core.setFailed(`Post step failed: ${e}`);
}
