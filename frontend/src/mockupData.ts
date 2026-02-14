import { DateTime, Duration } from "luxon";
import { PRICE_RELEASE_TIME } from "./hooks/useElectricityPrices";

// Toggle this to force stale timestamps and trigger outdated indicators in UI components.
export const MOCK_FORCE_OUTDATED_TIMESTAMPS = false;

const OUTDATED_THRESHOLDS: Record<string, Duration> = {
  "/api/weathernow": Duration.fromObject({ hours: 1 }),
  "/api/indoor/dev_upstairs": Duration.fromObject({ hours: 12 }),
  "/api/indoor/Shelly": Duration.fromObject({ hours: 12 }),
  "/api/cabinbookings/days/365": Duration.fromObject({ days: 1 }),
};

function timestampForEndpoint(
  path: string,
  freshOffset: Duration = Duration.fromObject({ minutes: 0 }),
): string {
  const now = DateTime.now().toUTC();
  const threshold = OUTDATED_THRESHOLDS[path];

  if (MOCK_FORCE_OUTDATED_TIMESTAMPS && threshold) {
    return now
      .minus(threshold)
      .minus(Duration.fromObject({ minutes: 5 }))
      .minus(freshOffset)
      .toISO();
  }

  return now.minus(freshOffset).toISO();
}

function formatSunTime(time: DateTime): string {
  return time.toFormat("h:mm:ss a");
}

function generateCabinBookings() {
  const startDate = DateTime.now()
    .toUTC()
    .minus({ days: 365 / 2 })
    .set({ hour: 16, minute: 0, second: 0, millisecond: 0 });

  const bookings = [];
  for (let i = 0; i < 365; i++) {
    const date = startDate.plus({ days: i });
    bookings.push({
      date: date.toISO(),
      booked: Math.random() > 0.7,
      updated: date.minus({ days: Math.floor(Math.random() * 60) + 1 }).toISO(),
    });
  }

  return {
    bookings,
    lastupdated: timestampForEndpoint("/api/cabinbookings/days/365"),
  };
}

export function getMockData(path: string): object | undefined {
  switch (path) {
    case "/api/electricity/current":
      return {
        datetime: DateTime.now().toUTC().toISO(),
        powerw: 2500,
      };

    case "/api/indoor/dev_upstairs":
      return {
        battery: 100.0,
        humidity: 27.4,
        temperature: 22.5,
        time: timestampForEndpoint("/api/indoor/dev_upstairs"),
      };

    case "/api/weathernow": {
      const latest = DateTime.fromISO(timestampForEndpoint("/api/weathernow"));
      return [4, 3, 2, 1, 0].map((stepsAgo) => ({
        datetime: latest.minus({ minutes: stepsAgo * 10 }).toISO(),
        temperature: Math.round((Math.random() * 20 - 10) * 10) / 10,
        humidity: Math.round(Math.random() * 1000) / 10,
      }));
    }

    case "/api/indoor/Shelly":
      return {
        battery: 92.0,
        humidity: 80.5,
        temperature: -3.5,
        time: timestampForEndpoint("/api/indoor/Shelly"),
      };

    case "/api/electricity/prices": {
      const lowestPrice = 0;
      const highestPrice = 20;
      const prices = [];
      const dateTimeNowEven = DateTime.now().set({
        minute: 0,
        second: 0,
        millisecond: 0,
      });
      const lastPriceTime =
        DateTime.now().hour >= PRICE_RELEASE_TIME.hour &&
        DateTime.now().minute >= PRICE_RELEASE_TIME.minute
          ? dateTimeNowEven.plus({ days: 1 })
          : dateTimeNowEven.set({ hour: 23 });

      for (
        let i = dateTimeNowEven.minus({ hours: 5 });
        i <= lastPriceTime;
        i = i.plus({ minutes: 15 })
      ) {
        prices.push({
          DateTime: i.toISO(),
          Price: Math.random() * (highestPrice - lowestPrice) + lowestPrice,
        });
      }
      return prices;
    }

    case "/api/outdoor/now":
      return [
        {
          temperature: -5.7,
          time: DateTime.now().toUTC().toISO(),
        },
      ];

    case "/api/outdoor/history/Kumpula/30": {
      const data = [];
      const now = DateTime.now();
      for (let i = 0; i < 30; i++) {
        const date = now.minus({ days: i });
        data.push({
          dt: date.toISODate(),
          t2m: Math.round((Math.random() * 10 - 5) * 10) / 10,
          r_1h: Math.round(Math.random() * 5 * 10) / 10,
        });
      }
      return data;
    }

    case "/api/weatherfore": {
      const data = [];
      const now = DateTime.now();
      for (let i = 0; i < 48; i++) {
        const date = now.plus({ hours: i });
        data.push({
          datetime: date.toISO(),
          weather: i % 5 === 0 ? "1" : "2",
          temperature: Math.round((Math.random() * 10 - 5) * 10) / 10,
          wind_dir: Math.round(Math.random() * 360),
          wind_speed: Math.round(Math.random() * 10),
          rain: Math.round(Math.random() * 5 * 10) / 10,
        });
      }
      return data;
    }

    case "/api/cabinbookings/days/365":
      return generateCabinBookings();

    case "/api/events": {
      const events = [];
      const now = DateTime.now();
      const eventTypes = [
        "Family dinner",
        "Elise's soccer",
        "Elias's hockey",
        "Ella's dance",
        "aiti's meeting",
        "iska's work trip",
      ];

      for (let i = 0; i < 10; i++) {
        const date = now.plus({ days: Math.floor(i / 2) });
        const startHour = 8 + Math.floor(Math.random() * 12);
        const endHour = startHour + 1 + Math.floor(Math.random() * 3);

        const start = date.set({
          hour: startHour,
          minute: 0,
          second: 0,
          millisecond: 0,
        });

        const end = date.set({
          hour: endHour,
          minute: 0,
          second: 0,
          millisecond: 0,
        });

        events.push({
          uid: `event-${i}`,
          summary: eventTypes[i % eventTypes.length],
          start: start.toISO(),
          end: end.toISO(),
        });
      }
      return events;
    }

    case "/api/sun": {
      const today = DateTime.now();
      const tomorrow = today.plus({ days: 1 });
      const dayAfterTomorrow = today.plus({ days: 2 });

      return [
        {
          date: today.toISODate(),
          sunrise: formatSunTime(
            today.set({ hour: 6, minute: 23, second: 21 }),
          ),
          sunset: formatSunTime(
            today.set({ hour: 18, minute: 34, second: 44 }),
          ),
          first_light: formatSunTime(
            today.set({ hour: 3, minute: 56, second: 42 }),
          ),
          last_light: formatSunTime(
            today.set({ hour: 21, minute: 1, second: 22 }),
          ),
          dawn: formatSunTime(today.set({ hour: 5, minute: 41, second: 33 })),
          dusk: formatSunTime(today.set({ hour: 19, minute: 16, second: 32 })),
          solar_noon: formatSunTime(
            today.set({ hour: 12, minute: 29, second: 2 }),
          ),
          golden_hour: formatSunTime(
            today.set({ hour: 17, minute: 39, second: 29 }),
          ),
          day_length: "12:11:22",
          timezone: "Europe/Helsinki",
          utc_offset: 120,
        },
        {
          date: tomorrow.toISODate(),
          sunrise: formatSunTime(
            tomorrow.set({ hour: 6, minute: 20, second: 17 }),
          ),
          sunset: formatSunTime(
            tomorrow.set({ hour: 18, minute: 37, second: 10 }),
          ),
          first_light: formatSunTime(
            tomorrow.set({ hour: 3, minute: 52, second: 52 }),
          ),
          last_light: formatSunTime(
            tomorrow.set({ hour: 21, minute: 4, second: 35 }),
          ),
          dawn: formatSunTime(
            tomorrow.set({ hour: 5, minute: 38, second: 25 }),
          ),
          dusk: formatSunTime(
            tomorrow.set({ hour: 19, minute: 19, second: 2 }),
          ),
          solar_noon: formatSunTime(
            tomorrow.set({ hour: 12, minute: 28, second: 44 }),
          ),
          golden_hour: formatSunTime(
            tomorrow.set({ hour: 17, minute: 41, second: 59 }),
          ),
          day_length: "12:16:53",
          timezone: "Europe/Helsinki",
          utc_offset: 120,
        },
        {
          date: dayAfterTomorrow.toISODate(),
          sunrise: formatSunTime(
            dayAfterTomorrow.set({ hour: 6, minute: 18, second: 10 }),
          ),
          sunset: formatSunTime(
            dayAfterTomorrow.set({ hour: 18, minute: 39, second: 22 }),
          ),
          first_light: formatSunTime(
            dayAfterTomorrow.set({ hour: 3, minute: 50, second: 23 }),
          ),
          last_light: formatSunTime(
            dayAfterTomorrow.set({ hour: 21, minute: 6, second: 58 }),
          ),
          dawn: formatSunTime(
            dayAfterTomorrow.set({ hour: 5, minute: 36, second: 17 }),
          ),
          dusk: formatSunTime(
            dayAfterTomorrow.set({ hour: 19, minute: 20, second: 32 }),
          ),
          solar_noon: formatSunTime(
            dayAfterTomorrow.set({ hour: 12, minute: 28, second: 20 }),
          ),
          golden_hour: formatSunTime(
            dayAfterTomorrow.set({ hour: 17, minute: 44, second: 25 }),
          ),
          day_length: "12:23:08",
          timezone: "Europe/Helsinki",
          utc_offset: 120,
        },
      ];
    }

    default:
      return undefined;
  }
}
