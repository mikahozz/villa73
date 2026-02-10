import { DateTime } from "luxon";
import dummyBookings from "./components/dummy-bookings";
import { PRICE_RELEASE_TIME } from "./hooks/useElectricityPrices";

export function getMockData(path: string): object | undefined {
  // Route handlers
  switch (path) {
    case "/api/electricity/current":
      return {
        datetime: "2022-04-02T11:55:58.103Z",
        powerw: 2500,
      };

    case "/api/indoor/dev_upstairs":
      return {
        battery: 100.0,
        humidity: 27.4,
        temperature: 22.5,
        time: "2022-01-31T19:06:06.604000Z",
      };

    case "/api/weathernow":
      return [
        {
          datetime: "2022-01-31T18:40:05Z",
          temperature: -15.5,
          humidity: 7.0,
        },
        {
          datetime: "2022-01-31T18:50:05Z",
          temperature: -5.5,
          humidity: 7.0,
        },
        {
          datetime: "2022-01-31T19:00:05Z",
          temperature: -15.6,
          humidity: 7.0,
        },
        {
          datetime: "2022-01-31T19:10:05Z",
          temperature: -15.6,
          humidity: 7.0,
        },
        {
          datetime: "2022-01-31T16:20:05Z",
          temperature: -15.7,
          humidity: 7.0,
        },
      ];

    case "/api/indoor/Shelly":
      return {
        battery: 92.0,
        humidity: 80.5,
        temperature: -3.5,
        time: "2022-01-31T19:36:07.313000Z",
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
          : dateTimeNowEven.set({
              hour: 23,
            });

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
      return [{ temperature: -5.7, time: "2022-01-31T19:20:05Z" }];

    case "/api/outdoor/history/Kumpula/30": {
      // Generate mock weather history data
      const data = [];
      const now = new Date();
      for (let i = 0; i < 30; i++) {
        const date = new Date();
        date.setDate(now.getDate() - i);
        data.push({
          dt: date.toISOString().split("T")[0],
          t2m: Math.round((Math.random() * 10 - 5) * 10) / 10,
          r_1h: Math.round(Math.random() * 5 * 10) / 10,
        });
      }
      return data;
    }

    case "/api/weatherfore": {
      // Generate mock forecast data
      const data = [];
      const now = new Date();
      for (let i = 0; i < 48; i++) {
        const date = new Date();
        date.setHours(now.getHours() + i);
        data.push({
          datetime: date.toISOString(),
          weather: i % 5 === 0 ? "1" : "2",
          temperature: Math.round((Math.random() * 10 - 5) * 10) / 10,
          wind_dir: Math.round(Math.random() * 360),
          wind_speed: Math.round(Math.random() * 10),
          rain: Math.round(Math.random() * 5 * 10) / 10,
        });
      }
      return data;
    }

    case "/api/cabinbookings/days/365": {
      return dummyBookings;
      // Generate mock cabin booking data
      const bookings = [];
      const startDate = DateTime.now()
        .toUTC()
        .minus({ days: 365 / 2 })
        .set({ hour: 16, minute: 0, second: 0, millisecond: 0 });
      for (let i = 0; i < 365; i++) {
        const newDate = startDate.plus({ days: i });
        bookings.push({
          date: newDate.toISO(),
          booked: Math.random() > 0.7,
          updated: newDate
            .minus({ days: Math.floor(Math.random() * 60) + 1 })
            .toISO(),
        });
      }
      return {
        bookings,
        lastupdated: DateTime.now().toISO(),
      };
    }

    case "/api/events": {
      // Generate mock calendar events
      const events = [];
      const now = new Date();
      const eventTypes = [
        "Family dinner",
        "Elise's soccer",
        "Elias's hockey",
        "Ella's dance",
        "äiti's meeting",
        "iskä's work trip",
      ];

      for (let i = 0; i < 10; i++) {
        const date = new Date();
        date.setDate(now.getDate() + Math.floor(i / 2));
        const startHour = 8 + Math.floor(Math.random() * 12);
        const endHour = startHour + 1 + Math.floor(Math.random() * 3);

        const start = new Date(date);
        start.setHours(startHour, 0, 0);

        const end = new Date(date);
        end.setHours(endHour, 0, 0);

        events.push({
          uid: `event-${i}`,
          summary: eventTypes[i % eventTypes.length],
          start: start.toISOString(),
          end: end.toISOString(),
        });
      }
      return events;
    }

    case "/api/sun": {
      // Create sun data for today and tomorrow directly
      const today = new Date();
      const tomorrow = new Date(today);
      tomorrow.setDate(today.getDate() + 1);
      const dayAfterTomorrow = new Date(today);
      dayAfterTomorrow.setDate(today.getDate() + 2);
      // Format dates as YYYY-MM-DD
      const todayFormatted = today.toISOString().split("T")[0];
      const tomorrowFormatted = tomorrow.toISOString().split("T")[0];
      const dayAfterTomorrowFormatted = dayAfterTomorrow
        .toISOString()
        .split("T")[0];
      // Create the sun data directly
      return [
        {
          date: todayFormatted,
          sunrise: "6:23:21 AM",
          sunset: "6:34:44 PM",
          first_light: "3:56:42 AM",
          last_light: "9:01:22 PM",
          dawn: "5:41:33 AM",
          dusk: "7:16:32 PM",
          solar_noon: "12:29:02 PM",
          golden_hour: "5:39:29 PM",
          day_length: "12:11:22",
          timezone: "Europe/Helsinki",
          utc_offset: 120,
        },
        {
          date: tomorrowFormatted,
          sunrise: "6:20:17 AM",
          sunset: "6:37:10 PM",
          first_light: "3:52:52 AM",
          last_light: "9:04:35 PM",
          dawn: "5:38:25 AM",
          dusk: "7:19:02 PM",
          solar_noon: "12:28:44 PM",
          golden_hour: "5:41:59 PM",
          day_length: "12:16:53",
          timezone: "Europe/Helsinki",
          utc_offset: 120,
        },
        {
          date: dayAfterTomorrowFormatted,
          sunrise: "6:18:10 AM",
          sunset: "6:39:22 PM",
          first_light: "3:50:23 AM",
          last_light: "9:06:58 PM",
          dawn: "5:36:17 AM",
          dusk: "7:20:32 PM",
          solar_noon: "12:28:20 PM",
          golden_hour: "5:44:25 PM",
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
