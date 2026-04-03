import { describe, it, expect } from "vitest";
import { parseTimeInput } from "./timeparse";

// Fixed reference time: 2026-04-03 14:30:00
const NOW = new Date(2026, 3, 3, 14, 30, 0); // April 3, 2026 local

describe("parseTimeInput", () => {
  // --- Keywords ---

  describe("keywords", () => {
    it("empty string → all time", () => {
      expect(parseTimeInput("", NOW)).toEqual({ from: null, to: null, label: "All time" });
    });

    it('"all" → all time', () => {
      expect(parseTimeInput("all", NOW)).toEqual({ from: null, to: null, label: "All time" });
    });

    it('"all time" → all time', () => {
      expect(parseTimeInput("All Time", NOW)).toEqual({ from: null, to: null, label: "All time" });
    });

    it('"today" → start of today', () => {
      const r = parseTimeInput("today", NOW)!;
      expect(r).not.toBeNull();
      expect(r.from!.getDate()).toBe(NOW.getDate());
    });

    it('"yesterday" → previous day', () => {
      const r = parseTimeInput("yesterday", NOW)!;
      expect(r).not.toBeNull();
      expect(r.from!.getDate()).toBe(NOW.getDate() - 1);
    });

    it('"this month" → current month', () => {
      const r = parseTimeInput("this month", NOW)!;
      expect(r).not.toBeNull();
      expect(r.from!.getMonth()).toBe(3); // April
    });

    it('"last month" → previous month', () => {
      const r = parseTimeInput("last month", NOW)!;
      expect(r).not.toBeNull();
      expect(r.from!.getMonth()).toBe(2); // March
    });

    it('"this week" → current week', () => {
      const r = parseTimeInput("this week", NOW)!;
      expect(r).not.toBeNull();
      expect(r.from).toBeTruthy();
    });
  });

  // --- Shorthand relative ---

  describe("shorthand relative", () => {
    it('"45m" → 45 minutes ago', () => {
      const r = parseTimeInput("45m", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(45 * 60 * 1000, -3);
      expect(r.to).toBeNull();
    });

    it('"12h" → 12 hours ago', () => {
      const r = parseTimeInput("12h", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(12 * 60 * 60 * 1000, -3);
    });

    it('"10d" → 10 days ago', () => {
      const r = parseTimeInput("10d", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(10 * 24 * 60 * 60 * 1000, -3);
    });

    it('"2w" → 2 weeks ago', () => {
      const r = parseTimeInput("2w", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(14 * 24 * 60 * 60 * 1000, -3);
    });

    it('"3mo" → ~3 months ago', () => {
      const r = parseTimeInput("3mo", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(90 * 24 * 60 * 60 * 1000, -3);
    });

    it('"30s" → 30 seconds ago', () => {
      const r = parseTimeInput("30s", NOW)!;
      const diff = NOW.getTime() - r.from!.getTime();
      expect(diff).toBeCloseTo(30 * 1000, -3);
    });
  });

  // --- Natural language relative (chrono) ---

  describe("natural language relative", () => {
    it('"last 5 minutes" → 5 min ago', () => {
      const r = parseTimeInput("last 5 minutes", NOW)!;
      expect(r).not.toBeNull();
      expect(r.from!.getTime()).toBeLessThan(NOW.getTime());
    });

    it('"last 2 hours" → 2 hours ago', () => {
      const r = parseTimeInput("last 2 hours", NOW)!;
      expect(r).not.toBeNull();
    });

    it('"last 1 day" → 1 day ago', () => {
      const r = parseTimeInput("last 1 day", NOW)!;
      expect(r).not.toBeNull();
    });

    it('"last 3 weeks" → 3 weeks ago', () => {
      const r = parseTimeInput("last 3 weeks", NOW)!;
      expect(r).not.toBeNull();
    });

    it("preserves original label casing", () => {
      const r = parseTimeInput("Last 5 Minutes", NOW)!;
      expect(r.label).toBe("Last 5 Minutes");
    });
  });

  // --- Fixed single date ---

  describe("fixed single date", () => {
    it('"Apr 1" → April 1', () => {
      const r = parseTimeInput("Apr 1", NOW)!;
      expect(r.from!.getMonth()).toBe(3);
      expect(r.from!.getDate()).toBe(1);
      expect(r.to!.getHours()).toBe(23);
    });

    it('"April 4" → full month name', () => {
      const r = parseTimeInput("April 4", NOW)!;
      expect(r.from!.getMonth()).toBe(3);
      expect(r.from!.getDate()).toBe(4);
    });

    it('"oct 17" → October 17', () => {
      const r = parseTimeInput("oct 17", NOW)!;
      expect(r.from!.getMonth()).toBe(9);
      expect(r.from!.getDate()).toBe(17);
    });

    it('"October 17" → October 17', () => {
      const r = parseTimeInput("October 17", NOW)!;
      expect(r.from!.getMonth()).toBe(9);
      expect(r.from!.getDate()).toBe(17);
    });

    it("single date has to at end of day", () => {
      const r = parseTimeInput("Apr 1", NOW)!;
      expect(r.to!.getHours()).toBe(23);
      expect(r.to!.getMinutes()).toBe(59);
    });
  });

  // --- Fixed range ---

  describe("fixed date range", () => {
    it('"Apr 1 - Apr 2" → two-day range', () => {
      const r = parseTimeInput("Apr 1 - Apr 2", NOW)!;
      expect(r.from!.getDate()).toBe(1);
      expect(r.to!.getDate()).toBe(2);
      expect(r.to!.getHours()).toBe(23);
    });

    it('"April 1 - April 3" → full month names', () => {
      const r = parseTimeInput("April 1 - April 3", NOW)!;
      expect(r.from!.getDate()).toBe(1);
      expect(r.to!.getDate()).toBe(3);
    });

    it("range with em dash", () => {
      const r = parseTimeInput("Apr 1 – Apr 3", NOW)!;
      expect(r).not.toBeNull();
      expect(r.from).toBeTruthy();
      expect(r.to).toBeTruthy();
    });
  });

  // --- Growing (since) ---

  describe("growing / since", () => {
    it('"since April 4" → April 4 → now', () => {
      const r = parseTimeInput("since April 4", NOW)!;
      expect(r.from!.getMonth()).toBe(3);
      expect(r.from!.getDate()).toBe(4);
      expect(r.to).toBeNull();
    });

    it('"since april 4" → case insensitive', () => {
      const r = parseTimeInput("since april 4", NOW)!;
      expect(r.from!.getMonth()).toBe(3);
      expect(r.from!.getDate()).toBe(4);
    });

    it('"since oct 17" → October 17', () => {
      const r = parseTimeInput("since oct 17", NOW)!;
      expect(r.from!.getMonth()).toBe(9);
      expect(r.from!.getDate()).toBe(17);
    });

    it('"since yesterday" → start of yesterday → now', () => {
      const r = parseTimeInput("since yesterday", NOW)!;
      expect(r.from!.getDate()).toBe(NOW.getDate() - 1);
      expect(r.to).toBeNull();
    });

    it('"since Apr 1" → April 1 → now', () => {
      const r = parseTimeInput("since Apr 1", NOW)!;
      expect(r.from!.getMonth()).toBe(3);
      expect(r.from!.getDate()).toBe(1);
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
  });
});
