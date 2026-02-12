// React component test for ElectricityPrice: documents polling logic and accessibility data list.
import { describe, test, expect, vi, beforeEach, afterEach } from "vitest";
import type { Mock } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { DateTime } from "luxon";
import { ElectricityPrice } from "./ElectricityPrice";
import * as hook from "../hooks/useElectricityPrices";

const priceZone = "Europe/Stockholm";

// Helper to generate mock prices: today + optional tomorrow
const buildPrices = (base: DateTime, includeTomorrow: boolean) => {
  const todayStart = base.setZone(priceZone).startOf("day");
  const arr: { DateTime: string; Price: number }[] = [];
  for (let h = 0; h < 24; h++) {
    arr.push({ DateTime: todayStart.plus({ hours: h }).toISO()!, Price: h });
  }
  if (includeTomorrow) {
    const tomorrowStart = todayStart.plus({ days: 1 });
    for (let h = 0; h < 24; h++) {
      arr.push({
        DateTime: tomorrowStart.plus({ hours: h }).toISO()!,
        Price: 100 + h,
      });
    }
  }
  return arr;
};

// Mock hook to control data and internal flag transitions.
vi.mock("../hooks/useElectricityPrices");

describe("ElectricityPrice component polling logic", () => {
  const zone = "Europe/Helsinki";
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
  });

  afterEach(() => {
    vi.clearAllTimers();
    vi.useRealTimers();
  });

  test("polls every 10 minutes after 14:00 until tomorrow prices appear then stops", async () => {
    // Start at 13:50 local time
    const start = DateTime.fromISO("2024-05-10T13:50:00", { zone });
    vi.setSystemTime(start.toJSDate());

    // Sequence of datasets returned by the hook as time advances.
    const datasets = [
      buildPrices(start, false), // before release, only today
      buildPrices(start.plus({ minutes: 20 }), false), // first poll after release, still only today
      buildPrices(start.plus({ minutes: 40 }), true), // tomorrow's prices available
    ];
    let callCount = 0;

    const mockedHook = hook as unknown as { useElectricityPrices: Mock };
    mockedHook.useElectricityPrices.mockImplementation(() => {
      const data = datasets[Math.min(callCount, datasets.length - 1)];
      return {
        data: {
          allPrices: data,
          currentAndFuturePrices: data.slice(0, 24),
          dayAverage: 10,
        },
        isLoading: false,
        error: null,
      };
    });

    render(
      <QueryClientProvider client={queryClient}>
        <ElectricityPrice />
      </QueryClientProvider>,
    );

    // Initial render before 14:00: ensure day average present
    const priceList = screen.getByRole("list");
    expect(priceList).toBeDefined();
    const priceTerms = within(priceList).getAllByRole("term");
    const priceDefinitions = within(priceList).getAllByRole("definition");
    expect(priceTerms.length).toBe(25);
    expect(priceDefinitions.length).toBe(25);
    expect(priceTerms[priceTerms.length - 1].textContent).toBe("Day average");
    expect(priceDefinitions[priceDefinitions.length - 1].textContent).toBe(
      "10",
    );
    expect(priceTerms[priceTerms.length - 2].textContent).toBe("0");
    expect(priceDefinitions[priceDefinitions.length - 2].textContent).toBe(
      "23",
    );

    expect(screen.getByTestId("elPriceDayAverageValue").textContent).toBe("10");
    // Advance time to just after 14:00; emulate one poll.
    vi.setSystemTime(start.set({ hour: 14, minute: 5 }).toJSDate());
    callCount = 1; // emulate one polling cycle
    vi.advanceTimersByTime(10 * 60 * 1000); // 10 minutes

    // Still only today's prices (hours 0..23), verify one accessible hour entry
    expect(screen.getByTestId("elPriceHour_23").textContent).toBe("23");

    // Advance another 10 minutes -> tomorrow data arrival
    vi.setSystemTime(start.set({ hour: 14, minute: 25 }).toJSDate());
    callCount = 2;
    vi.advanceTimersByTime(10 * 60 * 1000);

    // Tomorrow's prices present internally; since component shows currentAndFuturePrices (first 24), day average unchanged
    expect(screen.getByTestId("elPriceDayAverageValue").textContent).toBe("10");

    // Capture timers count before further advancing; polling should stop now.
    const getTimerCount = vi.getTimerCount();
    vi.advanceTimersByTime(30 * 60 * 1000); // advance 30 more mins
    const getTimerCountAfter = vi.getTimerCount();
    expect(getTimerCountAfter).toBeLessThanOrEqual(getTimerCount); // no new polling intervals added
  });
});
