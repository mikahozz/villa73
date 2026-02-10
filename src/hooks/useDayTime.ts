import { useQuery } from "@tanstack/react-query";
import { DateTime } from "luxon";
import { useCallback } from "react";

export interface SunItem {
  date: string;
  sunrise: string;
  sunset: string;
  first_light: string;
  last_light: string;
}

export function useDayTime() {
  const {
    data: sunData = [],
    isLoading,
    error,
  } = useQuery<SunItem[]>({
    queryKey: ["sunData"],
    queryFn: async () => {
      const start = DateTime.now().toISODate();
      const end = DateTime.now().plus({ days: 3 }).toISODate();
      console.log("Fetching sun data");
      const response = await fetch(`/api/sun?start=${start}&end=${end}`);
      if (!response.ok) throw new Error("Failed to fetch sun data");
      return response.json();
    },
    refetchInterval: 1000 * 60 * 60, // 1 hour
  });

  // Helper to check if a given date is daytime
  const isDayTime = useCallback(
    (dateTime: Date) => {
      const daySunData = sunData.find((item) => {
        const itemDate = new Date(item.date);
        return itemDate.toDateString() === dateTime.toDateString();
      });
      if (!daySunData) {
        return false;
      }

      // Use Luxon to parse sunrise and sunset times
      const parseSunTime = (timeStr: string, dateStr: string) => {
        const fullDateTimeStr = `${dateStr} ${timeStr}`;
        const dateTime = DateTime.fromFormat(
          fullDateTimeStr,
          "yyyy-MM-dd h:mm:ss a"
        );

        // Convert to JavaScript Date object for comparison
        return dateTime.toJSDate();
      };

      const sunrise = parseSunTime(daySunData.sunrise, daySunData.date);
      const sunset = parseSunTime(daySunData.sunset, daySunData.date);

      return dateTime >= sunrise && dateTime <= sunset;
    },
    [sunData]
  );

  return { isDayTime, isLoading, error };
}
