import { useQuery } from "@tanstack/react-query";
import { DateTime } from "luxon";
import { z } from "zod";
import { ElectricityPrice, ElectricityPriceSchema } from "../types/electricity";
import { useState } from "react";

const ELECTRICITY_PRICES_KEY = "electricityPrices";
// Polling interval when waiting for tomorrow's prices after the daily release time.
const PREFETCH_INTERVAL = 10 * 60 * 1000; // 10 minutes in milliseconds
// Official Nord Pool day-ahead prices usually published ~14:00 local (Europe/Helsinki)
export const PRICE_RELEASE_TIME = { hour: 14, minute: 0 };

const fetchElectricityPrices = async (): Promise<ElectricityPrice[]> => {
  const start = DateTime.now()
    .minus({ days: 1 })
    .set({ hour: 0, minute: 0, second: 0, millisecond: 0 })
    .toUTC()
    .toISO({ suppressMilliseconds: true });

  const end = DateTime.now()
    .plus({ days: 1 })
    .set({ hour: 23, minute: 0, second: 0, millisecond: 0 })
    .toUTC()
    .toISO({ suppressMilliseconds: true });

  console.log("fetching electricity prices from", start, "to", end);
  const timeZone = "Europe/Helsinki";
  const response = await fetch(
    `/api/electricity/prices?start=${start}&end=${end}&timeFormat=${timeZone}`
  );

  const data = await response.json();

  // Validate the response data
  const validatedData = z.array(ElectricityPriceSchema).parse(data);
  return validatedData;
};

// const shouldRefetchData = (data: ElectricityPrice[]): boolean => {
//   if (!data.length) return true;

//   // Find the latest price entry
//   const latestPrice = data.reduce((latest, current) => {
//     return DateTime.fromISO(current.DateTime) >
//       DateTime.fromISO(latest.DateTime)
//       ? current
//       : latest;
//   }, data[0]);

//   const latestPriceDate = DateTime.fromISO(latestPrice.DateTime).setZone(
//     "Europe/Helsinki"
//   );
//   const now = DateTime.now().setZone("Europe/Helsinki");

//   // If the latest price is from today and it's past 13:00 CET, we should refetch
//   // to get tomorrow's prices
//   if (
//     latestPriceDate.hasSame(now, "day") &&
//     now.hour >= PRICE_RELEASE_TIME.hour &&
//     now.minute >= PRICE_RELEASE_TIME.minute
//   ) {
//     return true;
//   }

//   return false;
// };

export const useElectricityPrices = (firstTimeToShow: DateTime) => {
  const [tomorrowsPricesFetched, setTomorrowsPricesFetched] = useState(false);

  const zone = "Europe/Helsinki";
  const priceZone = "Europe/Stockholm";

  // Compute next release (next 14:00 local time) for staleTime so the query becomes stale
  // right when we expect new data.
  const now = DateTime.now().setZone(zone);
  const todayRelease = now.set({
    hour: PRICE_RELEASE_TIME.hour,
    minute: PRICE_RELEASE_TIME.minute,
    second: 0,
    millisecond: 0,
  });
  const nextRelease =
    now < todayRelease ? todayRelease : todayRelease.plus({ days: 1 });
  const staleTimeMs = Math.max(0, nextRelease.toMillis() - now.toMillis());

  const query = useQuery({
    queryKey: [ELECTRICITY_PRICES_KEY],
    queryFn: () =>
      fetchElectricityPrices().then((data) => {
        // Sort to be safe
        const sorted = [...data].sort(
          (a, b) =>
            DateTime.fromISO(a.DateTime).toMillis() -
            DateTime.fromISO(b.DateTime).toMillis()
        );

        // Detect if we have any price whose CET date is tomorrow
        const tomorrowLocal = now
          .setZone(priceZone)
          .plus({ days: 1 })
          .startOf("day");
        const hasTomorrow = sorted.some((p) =>
          DateTime.fromISO(p.DateTime)
            .setZone(priceZone)
            .hasSame(tomorrowLocal, "day")
        );
        setTomorrowsPricesFetched(hasTomorrow);
        return sorted;
      }),
    refetchInterval: () => {
      // Poll every 10 mins after release until tomorrow's prices appear.
      const localNow = DateTime.now().setZone(zone);
      const releasePassed =
        localNow >=
        localNow.set({
          hour: PRICE_RELEASE_TIME.hour,
          minute: PRICE_RELEASE_TIME.minute,
          second: 0,
          millisecond: 0,
        });
      const shouldRefetch = !tomorrowsPricesFetched && releasePassed;
      console.log(
        "tomorrowsPricesFetched",
        tomorrowsPricesFetched,
        "shouldRefetch",
        shouldRefetch
      );
      if (shouldRefetch) {
        return PREFETCH_INTERVAL; // Refetch every 10 mins until tomorrow's prices are available
      }
      return false; // Do not refetch
    },
    gcTime: 1000 * 60 * 60 * 24, // 24 hours,
    staleTime: staleTimeMs,
    refetchIntervalInBackground: true,
    select: (data) => {
      // Process the data to add computed fields or filter if needed
      return {
        allPrices: data,
        currentAndFuturePrices: data
          .filter((price) => {
            const priceDateTime = DateTime.fromISO(price.DateTime);
            return priceDateTime >= firstTimeToShow;
          })
          .slice(0, 24),
        dayAverage: calculateDayAverage(data),
      };
    },
  });

  return query;
};

function calculateDayAverage(data: ElectricityPrice[]): number {
  const todayData = data.filter((item) => {
    const priceDateTime = DateTime.fromISO(item.DateTime);
    return priceDateTime.hour >= 8 && priceDateTime.hour <= 24;
  });

  if (todayData.length === 0) return 0;

  const sum = todayData.reduce((acc, curr) => acc + curr.Price, 0);
  return sum / todayData.length;
}
