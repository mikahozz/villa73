import { useState, useEffect } from "react";
import _ from "lodash";
import moment from "moment";
import { Modal, ModalHeader, ModalBody } from "reactstrap";
import * as utils from "../Utils";

export interface Bookings {
  bookings: BookingItem[];
  lastupdated: string;
}

interface BookingItem {
  date: string;
  booked: boolean;
  updated?: string;
}

interface GroupedBookings {
  [key: string]: BookingItem[];
}

export interface YearMonthWeekBookings {
  [year: string]: {
    [month: string]: {
      [week: string]: BookingItem[];
    };
  };
}

export function CabinBookings() {
  const [modal, setModal] = useState(false);
  const [bookingsdata, setBookingsdata] = useState<GroupedBookings>({});
  const [bookingsyeardata, setBookingsyeardata] =
    useState<YearMonthWeekBookings>({});
  const [lastUpdated, setLastUpdated] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    populateData();
    // Refresh data every 60 minutes
    const intervalId = setInterval(populateData, 60 * 60 * 1000);

    // Cleanup on unmount
    return () => clearInterval(intervalId);
  }, []);

  const populateData = async () => {
    try {
      const response = await fetch("/api/cabinbookings/days/365");
      const data: Bookings = await response.json();
      // Create a date object for the start of today
      const today = new Date();
      today.setHours(0, 0, 0, 0);

      const startOfToday = new Date(today.setHours(0, 0, 0, 0));
      const grouped = _.chain(data.bookings)
        .filter(
          (element) =>
            new Date(element.date).getTime() >= startOfToday.getTime()
        )
        .groupBy((element) => utils.getYearWeekNumber(new Date(element.date)))
        .value();
      const groupedByMonthWeek = utils.groupByMonthWeek(data.bookings);

      setBookingsdata(grouped);
      setBookingsyeardata(
        groupedByMonthWeek as unknown as YearMonthWeekBookings
      );
      setLastUpdated(data.lastupdated);
      setLoading(false);
    } catch (error) {
      console.error("Failed to fetch cabin bookings:", error);
    }
  };

  const renderContents = (
    bookingsdata: GroupedBookings,
    lastUpdated: string
  ) => {
    return (
      <div className="bookingsContainer">
        <div className="bookingsTable">
          {Object.keys(bookingsdata)
            .slice(0, 5)
            .map((week) => (
              <div className="bookingWeekRow noWeek" key={week}>
                {bookingsdata[week].map((bookingitem) => (
                  <div
                    className={renderBookingClasses(bookingitem)}
                    key={bookingitem.date}
                    title={bookingitem.date}
                  >
                    {new Date(bookingitem.date).getDate()}
                  </div>
                ))}
              </div>
            ))}
        </div>
        <div className={renderUpdatedClasses(Date.parse(lastUpdated))}>
          {moment(lastUpdated).format("ddd HH:mm")}
        </div>
      </div>
    );
  };

  const renderYearContents = (bookingsdata: YearMonthWeekBookings) => {
    const monthNames = [
      "January",
      "February",
      "March",
      "April",
      "May",
      "June",
      "July",
      "August",
      "September",
      "October",
      "November",
      "December",
    ];

    return (
      <div className="bookingsContainer">
        {Object.keys(bookingsdata).map((year) => (
          <div key={year}>
            <h2>{year}</h2>
            {Object.keys(bookingsdata[year]).map((month) => (
              <div className="bookingsMonth" key={month}>
                <div className="bookingsTable">
                  <div className="bookingsMonthTitle">
                    {monthNames[parseInt(month)]}
                  </div>
                  {Object.keys(bookingsdata[year][month]).map((week) => (
                    <div className="bookingWeekRow" key={week}>
                      <div className="bookingBox weekNumber">{week}</div>
                      {bookingsdata[year][month][week].map((bookingitem) => (
                        <div
                          className={renderBookingClasses(bookingitem)}
                          key={bookingitem.date}
                          title={bookingitem.date}
                        >
                          {new Date(bookingitem.date).getDate()}
                        </div>
                      ))}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        ))}
      </div>
    );
  };

  const renderBookingClasses = (bookingItem: BookingItem) => {
    let cssClass = "bookingBox day" + new Date(bookingItem.date).getDay();
    if (bookingItem.booked) {
      cssClass += " booked";
    }

    if (bookingItem.updated) {
      const updatedDate = new Date(Date.parse(bookingItem.updated));
      const daysAgo =
        Math.abs(new Date().getTime() - updatedDate.getTime()) /
        (1000 * 60 * 60 * 24);
      if (daysAgo < 7) {
        cssClass += " updatedWithinWeek";
      } else if (daysAgo < 14) {
        cssClass += " updatedWithin2Weeks";
      }
    }
    return cssClass;
  };

  const renderUpdatedClasses = (date: number) => {
    const diff = Math.abs(new Date().getTime() - date);
    let cssClass = "bookingsUpdated";
    if (diff / (1000 * 60 * 60 * 24) > 1) {
      cssClass += " outdated";
    }
    return cssClass;
  };

  const toggle = () => {
    setModal(!modal);
  };

  const contents = loading ? (
    <p>
      <em>Loading...</em>
    </p>
  ) : (
    renderContents(bookingsdata, lastUpdated)
  );

  const yearContents = loading ? (
    <p>
      <em>Loading...</em>
    </p>
  ) : (
    renderYearContents(bookingsyeardata)
  );

  return (
    <div id="cabinBookings" onClick={toggle}>
      <h2 className="small">Cabin bookings</h2>
      {contents}
      <Modal className="cabinbookings-modal" isOpen={modal} toggle={toggle}>
        <ModalHeader toggle={toggle}>Cabin bookings, upcoming year</ModalHeader>
        <ModalBody>{yearContents}</ModalBody>
      </Modal>
    </div>
  );
}

CabinBookings.displayName = "CabinBookings";
