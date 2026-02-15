import { Forecast } from "./Forecast";
import { WeatherData } from "./WeatherData";
import { WeatherNow } from "./WeatherNow";
import { Indoor } from "./Indoor";
import { Balcony } from "./Balcony";
import { Time } from "./Time";
import { FamilyCalendar } from "./FamilyCalendar";
import { CabinBookings } from "./CabinBookings";
import { ElectricityPrice } from "./ElectricityPrice";
import { Solar } from "./Solar";
import ConsoleLog from "./ConsoleLog";

export function Home() {
  return (
    <div className="w-full px-4">
      <ConsoleLog />
      <div className="box flex flex-col gap-4 leading-8 sm:flex-row sm:flex-wrap sm:[&>*]:flex sm:[&>*]:justify-center sm:[&>*:first-child]:justify-start sm:[&>*:last-child]:justify-end">
        <div className="sm:flex-1">
          <WeatherNow />
        </div>
        <div className="sm:flex-1" id="indoorContainer">
          <Indoor />
          <Balcony />
        </div>
        <div className="sm:flex-1 flex-col">
          <ElectricityPrice />
          <Solar />
        </div>
        <div className="sm:flex-1">
          <CabinBookings />
        </div>
        <div className="sm:flex-1">
          <Time />
        </div>
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-12">
        <div className="sm:col-span-5">
          <Forecast />
        </div>
        <div className="sm:col-span-7">
          <FamilyCalendar />
        </div>
      </div>
      <div className="grid grid-cols-1 gap-4">
        <div className="col-span-1">
          <WeatherData />
        </div>
      </div>
    </div>
  );
}

// Preserve the displayName
Home.displayName = "Home";
