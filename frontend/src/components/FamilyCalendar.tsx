import { useState, useEffect } from "react";
import moment from "moment";
import _ from "lodash";

interface CalendarEvent {
  uid: string;
  summary: string;
  start: string;
  end: string;
}

interface GroupedEvents {
  [date: string]: CalendarEvent[];
}

export function FamilyCalendar() {
  const [calendardata, setCalendardata] = useState<GroupedEvents>({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    populateCalendarData();
    // Refresh once in 1h
    const intervalId = setInterval(populateCalendarData, 60 * 60 * 1000);

    // Cleanup on unmount
    return () => clearInterval(intervalId);
  }, []);

  const populateCalendarData = async () => {
    try {
      const response = await fetch("/api/events");
      const data = await response.json();
      const sorted = _.sortBy(
        data,
        (element: CalendarEvent) => new Date(element.start)
      );
      const grouped = _.groupBy(sorted, (element: CalendarEvent) =>
        new Date(element.start).setHours(0, 0, 0, 0).toString()
      );

      setCalendardata(grouped);
      setLoading(false);
    } catch (error) {
      console.error("Failed to fetch calendar data:", error);
    }
  };

  const renderCalendarContents = (calendardata: GroupedEvents) => {
    return Object.keys(calendardata).map((calitem) => (
      <div key={calitem}>
        <h3>{renderDate(calitem)}</h3>
        {calendardata[calitem].map((eventItem) => (
          <div
            key={eventItem.uid}
            className={renderEventClasses(eventItem.summary)}
          >
            <div className="eventTitle">
              {eventItem.summary} {renderDots(eventItem.summary)}
              <span className="eventTime">
                {moment(eventItem.start).format("HH:mm")} -{" "}
                {moment(eventItem.end).format("HH:mm")}
              </span>
            </div>
          </div>
        ))}
      </div>
    ));
  };

  const renderDots = (valueObject: string) => {
    const value = String(valueObject).toLowerCase();
    const classes = [];

    if (value.includes("elise")) {
      classes.push(<span key="elise" className="elise dot"></span>);
    }
    if (value.includes("elias") || value.includes("eliaksen")) {
      classes.push(<span key="elias" className="elias dot"></span>);
    }
    if (value.includes("ella")) {
      classes.push(<span key="ella" className="ella dot"></span>);
    }
    if (value.includes("äiti")) {
      classes.push(<span key="aiti" className="aiti dot"></span>);
    }
    if (value.includes("iskä")) {
      classes.push(<span key="iska" className="iska dot"></span>);
    }

    return classes;
  };

  const renderEventClasses = (value: string) => {
    return value[0] === "#" ? "calendarBox lowPrio" : "calendarBox";
  };

  const renderDate = (dateNumber: string) => {
    const weekDays = [
      "Sunday",
      "Monday",
      "Tuesday",
      "Wednesday",
      "Thursday",
      "Friday",
      "Saturday",
    ];
    const eventDate = new Date(Number(dateNumber)).setHours(0, 0, 0, 0);
    const todayDate = new Date();
    const tomorrowDate = new Date();
    tomorrowDate.setDate(todayDate.getDate() + 1);
    const today = todayDate.setHours(0, 0, 0, 0);
    const tomorrow = tomorrowDate.setHours(0, 0, 0, 0);

    switch (eventDate) {
      case today:
        return "Today";
      case tomorrow:
        return "Tomorrow";
      default:
        return weekDays[new Date(eventDate).getDay()];
    }
  };

  const contents = loading ? (
    <p>
      <em>Loading...</em>
    </p>
  ) : (
    renderCalendarContents(calendardata)
  );

  return (
    <div id="calendardata" className="box">
      <h2>Upcoming family events</h2>
      {contents}
    </div>
  );
}

FamilyCalendar.displayName = "FamilyCalendar";
