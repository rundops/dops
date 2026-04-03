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
 *
 * Supported formats:
 *
 * Keywords:
 *   "all", "all time"       → no filter
 *   "today"                 → start of today → now
 *   "yesterday"             → start of yesterday → start of today
 *   "last month"            → 30 days ago → now
 *
 * Relative (duration ago → now):
 *   "45m", "45 min", "45 minutes"
 *   "12h", "12 hours", "12 hr"
 *   "10d", "10 days"
 *   "2w", "2 weeks"
 *   "3mo", "3 months"
 *   "last 5 min", "last 2 hours"
 *
 * Fixed (date → end of date):
 *   "Apr 1"                 → Apr 1 00:00 → Apr 1 23:59
 *   "4/1"                   → Apr 1 (current year)
 *   "2026-04-01"            → full ISO date
 *
 * Fixed range (date → date):
 *   "Apr 1 - Apr 2"
 *   "4/1 - 4/2"
 *   "2026-04-01 - 2026-04-03"
 *
 * Growing (date → now):
 *   "since 4/1"
 *   "since yesterday"
 *   "since Apr 1"
 */
export function parseTimeInput(raw: string, now?: Date): TimeRange | null {
  const refNow = now ?? new Date();
  const s = raw.trim().toLowerCase();

  if (!s || s === "all" || s === "all time") {
    return { from: null, to: null, label: "All time" };
  }

  // Keywords
  if (s === "today") {
    const d = new Date(refNow);
    d.setHours(0, 0, 0, 0);
    return { from: d, to: null, label: "Today" };
  }
  if (s === "yesterday") {
    const from = new Date(refNow);
    from.setDate(from.getDate() - 1);
    from.setHours(0, 0, 0, 0);
    const to = new Date(refNow);
    to.setHours(0, 0, 0, 0);
    return { from, to, label: "Yesterday" };
  }
  if (s === "last month") {
    return {
      from: new Date(refNow.getTime() - 30 * 24 * 60 * 60 * 1000),
      to: null,
      label: "Last month",
    };
  }

  // "since <date>" or "since yesterday"
  const sinceMatch = s.match(/^since\s+(.+)$/);
  if (sinceMatch) {
    const rest = sinceMatch[1].trim();
    if (rest === "yesterday") {
      const d = new Date(refNow);
      d.setDate(d.getDate() - 1);
      d.setHours(0, 0, 0, 0);
      return { from: d, to: null, label: raw.trim() };
    }
    const d = parseDate(rest, refNow);
    if (d) return { from: d, to: null, label: raw.trim() };
  }

  // Relative: "45m", "12 hours", "last 5 min"
  const relMatch = s.match(
    /^(?:last\s+)?(\d+)\s*(s|sec|seconds?|m|min|minutes?|h|hr|hours?|d|days?|w|weeks?|mo|months?)$/
  );
  if (relMatch) {
    const n = parseInt(relMatch[1], 10);
    const unit = relMatch[2];
    const ms = unitToMs(unit, n);
    if (ms > 0) {
      return {
        from: new Date(refNow.getTime() - ms),
        to: null,
        label: raw.trim(),
      };
    }
  }

  // Fixed range: "Apr 1 - Apr 2", "4/1 - 4/2", "2026-04-01 - 2026-04-03"
  // Use " - " or " – " (with spaces) to avoid splitting ISO date hyphens.
  const rangeSep = s.match(/^(.+?)\s+[-–]\s+(.+)$/);
  if (rangeSep) {
    const from = parseDate(rangeSep[1].trim(), refNow);
    const to = parseDate(rangeSep[2].trim(), refNow);
    if (from && to) {
      to.setHours(23, 59, 59, 999);
      return { from, to, label: raw.trim() };
    }
  }

  // Single date: "Apr 1", "4/1", "2026-04-01"
  const d = parseDate(s, refNow);
  if (d) {
    const to = new Date(d);
    to.setHours(23, 59, 59, 999);
    return { from: d, to, label: raw.trim() };
  }

  return null;
}

function unitToMs(unit: string, n: number): number {
  if (unit.startsWith("mo")) return n * 30 * 24 * 60 * 60 * 1000;
  if (unit.startsWith("mi") || unit === "m") return n * 60 * 1000;
  if (unit.startsWith("s")) return n * 1000;
  if (unit.startsWith("h")) return n * 60 * 60 * 1000;
  if (unit.startsWith("d")) return n * 24 * 60 * 60 * 1000;
  if (unit.startsWith("w")) return n * 7 * 24 * 60 * 60 * 1000;
  return 0;
}

/**
 * Parse a date string, handling short forms like "Apr 1", "4/1" by
 * adding the reference year when the native parser fails.
 */
export function parseDate(input: string, refNow?: Date): Date | null {
  const now = refNow ?? new Date();
  const s = input.trim();
  if (!s) return null;

  // ISO format: "2026-04-01" → parse as local midnight (not UTC)
  const isoMatch = s.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (isoMatch) {
    const d = new Date(
      parseInt(isoMatch[1]),
      parseInt(isoMatch[2]) - 1,
      parseInt(isoMatch[3])
    );
    return isNaN(d.getTime()) ? null : d;
  }

  // M/D format: "4/1" → "4/1/2026"
  const slash = s.match(/^(\d{1,2})\/(\d{1,2})$/);
  if (slash) {
    const d = new Date(now.getFullYear(), parseInt(slash[1]) - 1, parseInt(slash[2]));
    return isNaN(d.getTime()) ? null : d;
  }

  // M/D/Y format: "4/1/2026"
  const slashYear = s.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})$/);
  if (slashYear) {
    const d = new Date(
      parseInt(slashYear[3]),
      parseInt(slashYear[1]) - 1,
      parseInt(slashYear[2])
    );
    return isNaN(d.getTime()) ? null : d;
  }

  // Month name patterns: "Apr 1", "March 15", "Apr 1, 2026"
  const monthMatch = s.match(
    /^(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)\w*\.?\s+(\d{1,2})(?:[,\s]+(\d{4}))?$/i
  );
  if (monthMatch) {
    const year = monthMatch[3] ? parseInt(monthMatch[3]) : now.getFullYear();
    const d = new Date(`${monthMatch[1]} ${monthMatch[2]}, ${year}`);
    return isNaN(d.getTime()) ? null : d;
  }

  return null;
}
