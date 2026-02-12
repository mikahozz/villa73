import { useState, useEffect } from "react";
import moment from "moment";

export function Time() {
  const [now, setNow] = useState(new Date());

  useEffect(() => {
    // Refresh data every 30s
    const intervalId = setInterval(() => {
      setNow(new Date());
    }, 30 * 1000);

    // Cleanup on unmount
    return () => clearInterval(intervalId);
  }, []);

  const weekDays = [
    "Sunday",
    "Monday",
    "Tuesday",
    "Wednesday",
    "Thursday",
    "Friday",
    "Saturday",
  ];

  return (
    <div id="time">
      <div className="time">{moment(now).format("HH:mm")}</div>
      <div className="date">
        {weekDays[now.getDay()]},{" "}
        {now.toLocaleString("fi-fi", { month: "short", day: "2-digit" })}
      </div>
    </div>
  );
}

Time.displayName = "Time";
