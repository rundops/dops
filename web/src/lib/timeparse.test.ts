import { describe, it, expect } from "vitest";
import { parseTimeInput, parseDate } from "./timeparse";

// Fixed reference time: 2026-04-03 14:30:00 UTC
const NOW = new Date("2026-04-03T14:30:00Z");

describe("parseTimeInput", () => {
  // --- Keywords ---

  describe("keywords", () => {
    it("empty string → all time", () => {
      const r = parseTimeInput("", NOW);
      expect(r).toEqual({ from: null, to: null, label: "All time" });
    });

    it('"all" → all time', () => {
      const r = parseTimeInput("all", NOW);
      expect(r).toEqual({ from: null, to: null, label: "All time" });
    });

    it('"all time" → all time', () => {
      const r = parseTimeInput("All Time", NOW);
      expect(r).toEqual({ from: null, to: null, label: "All time" });
    });

    it('"today" → start of today → now', () => {
      const r = parseTimeInput("today", NOW)!;
      expect(r.from!.getHours()).toBe(0);
      expect(r.from!.getMinutes()).toBe(0);
      expect(r.to).toBeNull(); // growing
      expect(r.label).toBe("Today");
    });

    it('"yesterday" → start of yesterday → start of today', () => {
      const r = parseTimeInput("yesterday", NOW)!;
      expect(r.from!.getDate()).toBe(NOW.getDate() - 1);
      expect(r.from!.getHours()).toBe(0);
      expect(r.to!.getHours()).toBe(0);
      expect(r.to!.getDate()).toBe(NOW.getDate());
      expect(r.label).toBe("Yesterday");
    });

    it('"last month" → 30 days ago → now', () => {
      const r = parseTimeInput("last month", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(30 * 24 * 60 * 60 * 1000, -3);
      expect(r.to).toBeNull();
    });
  });

  // --- Relative ---

  describe("relative durations", () => {
    it('"45m" → 45 minutes ago', () => {
      const r = parseTimeInput("45m", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(45 * 60 * 1000, -3);
      expect(r.to).toBeNull();
    });

    it('"45 min" → 45 minutes ago', () => {
      const r = parseTimeInput("45 min", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(45 * 60 * 1000, -3);
    });

    it('"45 minutes" → 45 minutes ago', () => {
      const r = parseTimeInput("45 minutes", NOW)!;
      expect(r.from).toBeTruthy();
    });

    it('"12h" → 12 hours ago', () => {
      const r = parseTimeInput("12h", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(12 * 60 * 60 * 1000, -3);
    });

    it('"12 hours" → 12 hours ago', () => {
      const r = parseTimeInput("12 hours", NOW)!;
      expect(r.from).toBeTruthy();
    });

    it('"12 hr" → 12 hours ago', () => {
      const r = parseTimeInput("12 hr", NOW)!;
      expect(r.from).toBeTruthy();
    });

    it('"10d" → 10 days ago', () => {
      const r = parseTimeInput("10d", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(10 * 24 * 60 * 60 * 1000, -3);
    });

    it('"10 days" → 10 days ago', () => {
      const r = parseTimeInput("10 days", NOW)!;
      expect(r.from).toBeTruthy();
    });

    it('"2w" → 2 weeks ago', () => {
      const r = parseTimeInput("2w", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(14 * 24 * 60 * 60 * 1000, -3);
    });

    it('"2 weeks" → 2 weeks ago', () => {
      const r = parseTimeInput("2 weeks", NOW)!;
      expect(r.from).toBeTruthy();
    });

    it('"3mo" → 3 months ago', () => {
      const r = parseTimeInput("3mo", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(90 * 24 * 60 * 60 * 1000, -3);
    });

    it('"3 months" → 3 months ago', () => {
      const r = parseTimeInput("3 months", NOW)!;
      expect(r.from).toBeTruthy();
    });

    it('"30s" → 30 seconds ago', () => {
      const r = parseTimeInput("30s", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(30 * 1000, -3);
    });

    it('"last 5 min" → 5 minutes ago', () => {
      const r = parseTimeInput("last 5 min", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(5 * 60 * 1000, -3);
    });

    it('"last 2 hours" → 2 hours ago', () => {
      const r = parseTimeInput("last 2 hours", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(2 * 60 * 60 * 1000, -3);
    });

    it('"last 1 day" → 1 day ago', () => {
      const r = parseTimeInput("last 1 day", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(24 * 60 * 60 * 1000, -3);
    });

    it("preserves original label", () => {
      const r = parseTimeInput("Last 5 Min", NOW)!;
      expect(r.label).toBe("Last 5 Min");
    });
  });

  // --- Fixed single date ---

  describe("fixed single date", () => {
    it('"Apr 1" → Apr 1 of current ref year', () => {
      const r = parseTimeInput("Apr 1", NOW)!;
      expect(r.from!.getMonth()).toBe(3); // April = 3
      expect(r.from!.getDate()).toBe(1);
      expect(r.to!.getHours()).toBe(23);
      expect(r.to!.getMinutes()).toBe(59);
    });

    it('"4/1" → Apr 1 of current ref year', () => {
      const r = parseTimeInput("4/1", NOW)!;
      expect(r.from!.getMonth()).toBe(3);
      expect(r.from!.getDate()).toBe(1);
    });

    it('"2026-04-01" → exact date', () => {
      const r = parseTimeInput("2026-04-01", NOW)!;
      expect(r.from!.getFullYear()).toBe(2026);
      expect(r.from!.getMonth()).toBe(3);
      expect(r.from!.getDate()).toBe(1);
    });

    it("single date has to at end of day", () => {
      const r = parseTimeInput("Apr 1", NOW)!;
      expect(r.to!.getHours()).toBe(23);
      expect(r.to!.getMinutes()).toBe(59);
      expect(r.to!.getSeconds()).toBe(59);
    });
  });

  // --- Fixed range ---

  describe("fixed date range", () => {
    it('"Apr 1 - Apr 2" → two-day range', () => {
      const r = parseTimeInput("Apr 1 - Apr 2", NOW)!;
      expect(r.from!.getMonth()).toBe(3);
      expect(r.from!.getDate()).toBe(1);
      expect(r.to!.getMonth()).toBe(3);
      expect(r.to!.getDate()).toBe(2);
      expect(r.to!.getHours()).toBe(23); // end of day
    });

    it('"4/1 - 4/2" → two-day range', () => {
      const r = parseTimeInput("4/1 - 4/2", NOW)!;
      expect(r.from!.getMonth()).toBe(3);
      expect(r.from!.getDate()).toBe(1);
      expect(r.to!.getDate()).toBe(2);
    });

    it('"2026-04-01 - 2026-04-03" → ISO range', () => {
      const r = parseTimeInput("2026-04-01 - 2026-04-03", NOW)!;
      expect(r.from!.getDate()).toBe(1);
      expect(r.to!.getDate()).toBe(3);
      expect(r.to!.getHours()).toBe(23);
    });

    it("range with em dash", () => {
      const r = parseTimeInput("Apr 1 – Apr 3", NOW)!;
      expect(r.from).toBeTruthy();
      expect(r.to).toBeTruthy();
    });
  });

  // --- Growing (since) ---

  describe("growing / since", () => {
    it('"since 4/1" → Apr 1 → now', () => {
      const r = parseTimeInput("since 4/1", NOW)!;
      expect(r.from!.getMonth()).toBe(3);
      expect(r.from!.getDate()).toBe(1);
      expect(r.to).toBeNull(); // growing to now
    });

    it('"since Apr 1" → Apr 1 → now', () => {
      const r = parseTimeInput("since Apr 1", NOW)!;
      expect(r.from!.getMonth()).toBe(3);
      expect(r.from!.getDate()).toBe(1);
      expect(r.to).toBeNull();
    });

    it('"since yesterday" → start of yesterday → now', () => {
      const r = parseTimeInput("since yesterday", NOW)!;
      expect(r.from!.getDate()).toBe(NOW.getDate() - 1);
      expect(r.from!.getHours()).toBe(0);
      expect(r.to).toBeNull();
    });

    it('"since 2026-03-01" → March 1 → now', () => {
      const r = parseTimeInput("since 2026-03-01", NOW)!;
      expect(r.from!.getMonth()).toBe(2); // March
      expect(r.to).toBeNull();
    });
  });

  // --- Invalid input ---

  describe("invalid input", () => {
    it("garbage returns null", () => {
      expect(parseTimeInput("asdfghjkl", NOW)).toBeNull();
    });

    it("partial word returns null", () => {
      expect(parseTimeInput("las", NOW)).toBeNull();
    });

    it("no number with unit returns null", () => {
      expect(parseTimeInput("minutes", NOW)).toBeNull();
    });
  });
});

describe("parseDate", () => {
  const NOW = new Date("2026-04-03T14:30:00Z");

  it('"Apr 1" → April 1 of ref year', () => {
    const d = parseDate("Apr 1", NOW)!;
    expect(d.getMonth()).toBe(3);
    expect(d.getDate()).toBe(1);
  });

  it('"4/1" → April 1 of ref year', () => {
    const d = parseDate("4/1", NOW)!;
    expect(d.getMonth()).toBe(3);
    expect(d.getDate()).toBe(1);
  });

  it('"12/25" → December 25 of ref year', () => {
    const d = parseDate("12/25", NOW)!;
    expect(d.getMonth()).toBe(11);
    expect(d.getDate()).toBe(25);
  });

  it('"2026-04-01" → exact ISO date', () => {
    const d = parseDate("2026-04-01", NOW)!;
    expect(d.getFullYear()).toBe(2026);
    expect(d.getMonth()).toBe(3);
  });

  it('"March 15" → March 15 of ref year', () => {
    const d = parseDate("March 15", NOW)!;
    expect(d.getMonth()).toBe(2);
    expect(d.getDate()).toBe(15);
  });

  it("invalid string returns null", () => {
    expect(parseDate("not a date", NOW)).toBeNull();
  });

  it("empty string returns null", () => {
    expect(parseDate("", NOW)).toBeNull();
  });
});
