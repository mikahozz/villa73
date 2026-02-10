import React, { useState, useEffect } from "react";
import moment from "moment";
import { DateTime } from "luxon";
import { useDayTime } from "../hooks/useDayTime";

interface ForecastItem {
  datetime: string;
  weather: string;
  temperature: number;
  wind_dir: number;
  wind_speed: number;
  rain: number;
}

export function Forecast() {
  const [forecastdata, setForecastdata] = useState<ForecastItem[]>([]);
  const [loading, setLoading] = useState(true);
  const { isDayTime } = useDayTime();

  useEffect(() => {
    populateForecastData();
    // Refresh data every 15 min
    const intervalId = setInterval(populateForecastData, 15 * 60 * 1000);

    // Cleanup on unmount
    return () => clearInterval(intervalId);
  }, []);

  const renderRotate = (degree: number) => {
    return { transform: `rotate(${degree}deg)` };
  };

  const renderWeatherContents = (forecastdata: ForecastItem[]) => {
    let previousDay: string | undefined;
    const today = DateTime.now().startOf("day");

    return (
      <div>
        <table className="forecastTable">
          <tbody>
            {forecastdata.map((forecastitem) => {
              const itemDate = new Date(forecastitem.datetime);
              const itemDay = itemDate.toDateString();
              const dayChanged = previousDay ? itemDay !== previousDay : false;
              const isDay = isDayTime(itemDate);
              previousDay = itemDay;

              // Get day label for the divider
              let dayLabel = "";
              if (dayChanged) {
                const forecastDateTime =
                  DateTime.fromJSDate(itemDate).startOf("day");
                const diffInDays = forecastDateTime.diff(today, "days").days;

                if (diffInDays < 1) {
                  dayLabel = "TODAY";
                } else if (diffInDays < 2) {
                  dayLabel = "TOMORROW";
                } else {
                  // Format the day name (e.g., "WEDNESDAY")
                  dayLabel = forecastDateTime.toFormat("EEEE").toUpperCase();
                }
              }

              return (
                <React.Fragment key={forecastitem.datetime}>
                  {dayChanged && (
                    <tr className="dayDivider">
                      <td colSpan={5}>
                        <h3>{dayLabel}</h3>
                      </td>
                    </tr>
                  )}
                  <tr className={isDay ? "day-row" : "night-row"}>
                    <td className="time-col">
                      {moment(forecastitem.datetime).format("HH:mm")}
                    </td>
                    <td>
                      <img
                        alt=""
                        width="55"
                        height="55"
                        src={`/img/${forecastitem.weather}.svg`}
                      />
                    </td>
                    <td className="temperature-col">
                      {Math.round(forecastitem.temperature)}°
                    </td>
                    <td>
                      <div className="wind-container">
                        <img
                          alt=""
                          style={renderRotate(forecastitem.wind_dir - 180)}
                          src="/img/arrow.svg"
                          width="40px"
                          height="40px"
                        />
                        <span className="wind-text">
                          {Math.round(forecastitem.wind_speed)}
                        </span>
                      </div>
                    </td>
                    <td>
                      <div
                        className="rainBox"
                        style={{ width: `${forecastitem.rain * 10}px` }}
                      ></div>
                    </td>
                  </tr>
                </React.Fragment>
              );
            })}
          </tbody>
        </table>
      </div>
    );
  };

  const populateForecastData = async () => {
    try {
      const response = await fetch("/api/weatherfore");
      const data = await response.json();
      setForecastdata(data);

      setLoading(false);
    } catch (error) {
      console.error("Failed to fetch forecast data:", error);
    }
  };

  const contents = loading ? (
    <p>
      <em>Loading...</em>
    </p>
  ) : (
    renderWeatherContents(forecastdata)
  );

  return (
    <div id="forecast" className="box">
      <h2>Forecast 24h</h2>
      <h3>Tapanila, Helsinki</h3>
      {contents}
    </div>
  );
}

Forecast.displayName = "Forecast";
