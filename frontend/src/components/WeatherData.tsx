import { useState, useEffect } from "react";
import {
  LabelList,
  Tooltip,
  Legend,
  ComposedChart,
  Bar,
  Line,
  CartesianGrid,
  XAxis,
  YAxis,
  ResponsiveContainer,
} from "recharts";
import moment from "moment";
import _ from "lodash";

interface WeatherDataItem {
  dt: string;
  t2m: number;
  r_1h: number;
}

interface MappedWeatherData {
  dt: string;
  t2m: number;
  r_1h: number;
}

interface LabelProps {
  x?: number | string;
  y?: number | string;
  value?: number | string;
}

interface LegendEntry {
  color?: string;
  value?: string;
  payload?: {
    strokeDasharray?: string | number;
    value?: string | number;
    dataKey?: string;
  };
}

type LegendFormatter = (
  value: string | undefined,
  entry: LegendEntry
) => React.JSX.Element;

type LabelContentType = (props: LabelProps) => React.JSX.Element;

export function WeatherData() {
  const [weatherdata, setWeatherdata] = useState<WeatherDataItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    populateWeatherData();
    // Refresh once in 24h
    const intervalId = setInterval(populateWeatherData, 24 * 60 * 60 * 1000);

    // Cleanup on unmount
    return () => clearInterval(intervalId);
  }, []);

  const renderCustomizedLabel = (props: LabelProps) => {
    const { x = 0, y = 0, value } = props;
    return (
      <g>
        <text
          x={x}
          y={y}
          dy={-4}
          fill={"#fff"}
          fontSize={10}
          textAnchor="middle"
        >
          {value}
        </text>
      </g>
    );
  };

  const legendMap = { t2m: "temperature (avg)", r_1h: "precipitation (mm)" };

  const renderCustomizedLegend = (
    value: string | undefined,
    entry: LegendEntry
  ) => {
    const { color } = entry;
    const legendValue = value || "";

    return (
      <span style={{ color }}>
        {legendMap[legendValue as keyof typeof legendMap] || ""}
      </span>
    );
  };

  const populateWeatherData = async () => {
    try {
      const response = await fetch("/api/outdoor/history/Kumpula/30");
      const data = await response.json();
      const sorted = _.sortBy(data, (element: WeatherDataItem) => element.dt);
      const grouped = _.groupBy(sorted, (element: WeatherDataItem) =>
        element.dt.substring(0, 10)
      );
      const mapped = _.map(grouped, (val: WeatherDataItem[], id: string) => ({
        dt: id,
        t2m: _.meanBy(val, "t2m"),
        r_1h: _.sumBy(val, "r_1h"),
      }));

      setWeatherdata(mapped as unknown as WeatherDataItem[]);
      setLoading(false);
    } catch (error) {
      console.error("Failed to fetch weather data:", error);
    }
  };

  const renderWeatherContents = (weatherdata: MappedWeatherData[]) => {
    return (
      <div>
        <ResponsiveContainer width="100%" height={200}>
          <ComposedChart
            data={weatherdata}
            margin={{ top: 10, right: 10, left: -30, bottom: 0 }}
          >
            <CartesianGrid strokeDasharray="3 3" vertical={false} />
            <XAxis
              dataKey="dt"
              tickFormatter={(value) => moment(value).format("DD")}
            />
            <YAxis yAxisId="left" orientation="left" />
            <YAxis yAxisId="right" orientation="right" />
            <Tooltip />
            <Legend formatter={renderCustomizedLegend as LegendFormatter} />
            <Bar
              yAxisId="right"
              dataKey="r_1h"
              barSize={20}
              fill="#0088FE"
              isAnimationActive={false}
            >
              <LabelList
                dataKey="r_1h"
                position="top"
                content={renderCustomizedLabel as unknown as LabelContentType}
              />
            </Bar>
            <Line
              yAxisId="left"
              type="monotone"
              dataKey="t2m"
              stroke="#FF8042"
              strokeWidth={2}
              isAnimationActive={false}
            >
              <LabelList
                dataKey="t2m"
                position="top"
                content={renderCustomizedLabel as unknown as LabelContentType}
              />
            </Line>
          </ComposedChart>
        </ResponsiveContainer>
      </div>
    );
  };

  const contents = loading ? (
    <p>
      <em>Loading...</em>
    </p>
  ) : (
    renderWeatherContents(weatherdata)
  );

  return (
    <div id="weatherdata" className="box">
      <h2>Daily temperature (avg) and precipitation amount this month</h2>
      {contents}
    </div>
  );
}

WeatherData.displayName = "WeatherData";
