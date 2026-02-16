import { useQuery } from "@tanstack/react-query";

export interface WeatherData {
  temperature: number;
  datetime: string;
}

export default function useWeatherNow() {
  const ONE_HOUR_MS = 60 * 60 * 1000;

  return useQuery<WeatherData[]>({
    queryKey: ["weatherNow"],
    queryFn: async () => {
      console.log("Fetching weathernow...");
      const response = await fetch("/api/weathernow");
      if (!response.ok) {
        throw new Error(`Failed to fetch weather now: ${response.statusText}`);
      }
      const data = await response.json();
      if (response.ok) {
        console.log("Weather now data fetched successfully with data:", data);
      }
      return data;
    },
    staleTime: ONE_HOUR_MS,
    refetchInterval: ONE_HOUR_MS,
    refetchIntervalInBackground: true,
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
    retry: Infinity,
  });
}
