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
      <div className="box grid grid-cols-1 gap-4 leading-8 sm:grid-cols-12">
        <div className="sm:col-span-4">
          <div className="flex flex-wrap items-start gap-4">
            <WeatherNow />
            <div id="indoorContainer">
              <Indoor />
              <Balcony />
            </div>
          </div>
        </div>
        <div className="sm:col-span-2">
          <ElectricityPrice />
          <Solar />
        </div>
        <div className="sm:col-span-2">
          <CabinBookings />
        </div>
        <div className="hidden items-center sm:flex sm:col-span-4 md:col-span-3 md:col-start-10">
          <div>
            <Time />
            <ConsoleLog />
          </div>
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
