import { useQuery } from "@tanstack/react-query";

export interface WeatherData {
  temperature: number;
  datetime: string;
}

export default function useWeatherNow() {
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
    staleTime: 60 * 60 * 1000, // 1 hour in ms
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
    retry: Infinity,
  });
}
