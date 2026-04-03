import * as chrono from "chrono-node";

/**
 * Parsed time range result.
 * - from=null, to=null means "all time"
 * - to=null means "from → now" (growing)
 */
export interface TimeRange {
  from: Date | null;
  to: Date | null;
  label: string;
}

/**
 * Parse a natural-language time input string into a date range.
 * Powered by chrono-node for broad format support.
 *
 * Supported formats include:
 *   "today", "yesterday", "this month", "last month"
 *   "last 5 minutes", "45m", "12 hours", "2 weeks"
 *   "since April 4", "since oct. 17", "since yesterday"
 *   "Apr 1", "April 4", "4/1", "2026-04-01"
 *   "Apr 1 - Apr 3", "4/1 - 4/3", "2026-04-01 - 2026-04-03"
 *   "all", "all time"
 */
export function parseTimeInput(raw: string, now?: Date): TimeRange | null {
  const refNow = now ?? new Date();
  const s = raw.trim();
  const lower = s.toLowerCase();

  if (!lower || lower === "all" || lower === "all time") {
    return { from: null, to: null, label: "All time" };
  }

  // "since <expression>" → parse the date part, open-ended to now
  const sinceMatch = lower.match(/^since\s+(.+)$/);
  if (sinceMatch) {
    const parsed = chrono.parseDate(sinceMatch[1], refNow);
    if (parsed) {
      return { from: parsed, to: null, label: s };
    }
  }

  // Shorthand relative: "45m", "12h", "10d", "2w", "3mo"
  const shorthand = lower.match(
    /^(?:last\s+)?(\d+)\s*(s|m|h|d|w|mo)$/
  );
  if (shorthand) {
    const n = parseInt(shorthand[1], 10);
    const unit = shorthand[2];
    const ms = shorthandToMs(unit, n);
    if (ms > 0) {
      return { from: new Date(refNow.getTime() - ms), to: null, label: s };
    }
  }

  // Range: "Apr 1 - Apr 3", "4/1 - 4/3" (require spaces around separator)
  const rangeSep = s.match(/^(.+?)\s+[-–]\s+(.+)$/);
  if (rangeSep) {
    const from = chrono.parseDate(rangeSep[1], refNow);
    const to = chrono.parseDate(rangeSep[2], refNow);
    if (from && to) {
      to.setHours(23, 59, 59, 999);
      return { from, to, label: s };
    }
  }

  // Let chrono handle everything else: "today", "yesterday", "this month",
  // "last 5 minutes", "last month", "April 4", "oct. 17", "2026-04-01", etc.
  const results = chrono.parse(lower, refNow);
  if (results.length > 0) {
    const r = results[0];
    const from = r.start.date();

    // If chrono found an end date, use it as a range
    if (r.end) {
      const to = r.end.date();
      to.setHours(23, 59, 59, 999);
      return { from, to, label: s };
    }

    // "this month", "this week" → chrono returns a start; use as growing range
    // Single date → treat as full day
    if (isExactDate(r)) {
      const to = new Date(from);
      to.setHours(23, 59, 59, 999);
      return { from, to, label: s };
    }

    // Relative/growing: "last 5 minutes", "yesterday", etc.
    return { from, to: null, label: s };
  }

  return null;
}

/** Check if chrono parsed an exact date (no time component implied) */
function isExactDate(result: chrono.ParsedResult): boolean {
  // If day is certain but hour is not, it's an exact date like "Apr 1"
  return (
    result.start.isCertain("day") &&
    result.start.isCertain("month") &&
    !result.start.isCertain("hour")
  );
}

function shorthandToMs(unit: string, n: number): number {
  switch (unit) {
    case "s": return n * 1000;
    case "m": return n * 60 * 1000;
    case "h": return n * 60 * 60 * 1000;
    case "d": return n * 24 * 60 * 60 * 1000;
    case "w": return n * 7 * 24 * 60 * 60 * 1000;
    case "mo": return n * 30 * 24 * 60 * 60 * 1000;
    default: return 0;
  }
}
