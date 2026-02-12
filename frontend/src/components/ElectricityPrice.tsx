import { useCallback, useEffect, useMemo, useState } from "react";
import { VictoryBar, VictoryChart, VictoryAxis, VictoryLine } from "victory";
import { Modal, ModalHeader, ModalBody } from "reactstrap";
import { useElectricityPrices } from "../hooks/useElectricityPrices";
import { DateTime, Duration } from "luxon";
export function ElectricityPrice() {
  const chartRefreshInterval = useMemo(
    () => Duration.fromObject({ minutes: 1 }),
    []
  );
  const thisHour = () => DateTime.now().startOf("hour");
  const nextHour = useCallback(
    () => DateTime.now().plus(chartRefreshInterval).startOf("hour"),
    [chartRefreshInterval]
  );
  const timeUntilNextHour = useCallback(
    () => nextHour().diffNow().toMillis(),
    [nextHour]
  );

  const [firstTimeToShow, setFirstTimeToShow] = useState(thisHour());
  const [modal, setModal] = useState(false);
  const { data, isLoading, error } = useElectricityPrices(firstTimeToShow);

  useEffect(() => {
    const activateRefresh = () => {
      const timeoutId = setTimeout(() => {
        setFirstTimeToShow(thisHour());
        console.log(
          "El: Setting first time firstTimeToShow to",
          thisHour().toISO()
        );

        const intervalId = setInterval(() => {
          setFirstTimeToShow(thisHour());
          console.log(
            `El: Setting firstTimeToShow to: ${thisHour()} with interval ${chartRefreshInterval}`
          );
        }, chartRefreshInterval.toMillis());

        return intervalId;
      }, timeUntilNextHour());

      return timeoutId;
    };

    let timeoutId = activateRefresh();

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        setFirstTimeToShow(thisHour());
        console.log(
          "El: visibilityState to visible. setFirstTimeToShow to:",
          thisHour().toISO()
        );
        timeoutId = activateRefresh();
      } else {
        console.log("El: visibilityState to hidden. Clearing timers.");
        clearInterval(timeoutId);
        clearTimeout(timeoutId);
      }
    };

    console.log("El: Adding visibilitychange event listener.");
    document.addEventListener("visibilitychange", handleVisibilityChange);

    // Cleanup function
    return () => {
      console.log("El: Cleaning up timers and event listeners.");
      clearInterval(timeoutId);
      clearTimeout(timeoutId);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [chartRefreshInterval, timeUntilNextHour]);

  const toggle = () => setModal(!modal);

  const chartTheme = {
    axis: {
      style: {
        bar: {
          fill: "#ffffff",
        },
        axis: {
          stroke: "none",
        },
        ticks: {
          stroke: "none",
          size: 5,
        },
        tickLabels: {
          fill: "white",
          fontSize: 35,
        },
        grid: {
          fill: "none",
          stroke: "none",
        },
      },
    },
    bar: {
      style: {
        data: {
          fill: "#00ff99",
        },
      },
    },
    line: {
      style: {
        data: {
          stroke: "#00ff00",
          strokeDasharray: "4, 8",
          strokeWidth: 2,
        },
      },
    },
  };

  if (isLoading) {
    return (
      <div>
        <p className="elPrice">
          <em>Loading...</em>
        </p>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div>
        <p className="elPrice">
          <em>Error loading electricity prices</em>
        </p>
      </div>
    );
  }

  const chartData = data.currentAndFuturePrices.map((item) => ({
    x: item.DateTime,
    y: item.Price,
  }));

  const maxPrice = Math.max(...chartData.map((item) => item.y));
  const baseDomain = 20;
  const chartHeightMultiplier = 5;
  const extraPrice = maxPrice > baseDomain ? maxPrice - baseDomain : 0;
  const chartHeight = 300 + chartHeightMultiplier * extraPrice;
  const domainMax = maxPrice > baseDomain ? maxPrice : baseDomain;

  const contents = (
    <div>
      <h2 className="small">Electricity price</h2>
      {/* Accessible data list for screen readers only */}
      <dl
        style={{
          position: "absolute",
          left: "-10000px",
          top: "auto",
          width: "1px",
          height: "1px",
          overflow: "hidden",
        }}
        role="list"
        aria-label="Electricity prices"
      >
        {data.currentAndFuturePrices.map((p) => {
          const dt = new Date(Date.parse(p.DateTime));
          const hour = dt.getHours();
          return (
            <div key={p.DateTime}>
              <dt data-testid={`elPriceHour_${hour}`}>{hour}</dt>
              <dd data-testid={`elPriceValue_${hour}`}>{p.Price}</dd>
            </div>
          );
        })}
        <div>
          <dt data-testid="elPriceDayAverageLabel">Day average</dt>
          <dd data-testid="elPriceDayAverageValue">{data.dayAverage}</dd>
        </div>
      </dl>
      <VictoryChart
        theme={chartTheme}
        domain={{ y: [0, domainMax] }}
        domainPadding={10}
        height={chartHeight}
        padding={{ top: 0, bottom: 32, left: 50, right: 50 }}
      >
        <VictoryBar
          data={chartData}
          barRatio={0.8}
          style={{
            data: {
              fill: ({ datum }) =>
                datum.y >= data.dayAverage ? "#FF0046" : "rgb(0,255,121)",
              fillOpacity: 0.9,
              strokeWidth: ({ datum }) => {
                return Date.parse(datum.x) === new Date().setMinutes(0, 0, 0)
                  ? 3
                  : 1;
              },
              stroke: ({ datum }) => {
                return Date.parse(datum.x) === new Date().setMinutes(0, 0, 0)
                  ? "rgb(255,255,255,0.8)"
                  : "none";
              },
            },
          }}
        />
        <VictoryAxis
          style={{
            ticks: {
              fill: "transparent",
              size: 5,
            },
            tickLabels: { fontSize: 30 },
          }}
          tickFormat={(t) => {
            const dt = new Date(Date.parse(t));
            if (dt.getMinutes() === 0) {
              return dt.getHours();
            } else {
              return "";
            }
          }}
        />
        <VictoryLine y={() => data.dayAverage} />
      </VictoryChart>

      <Modal
        style={{ maxWidth: "1440px", width: "95%" }}
        isOpen={modal}
        toggle={toggle}
      >
        <ModalHeader toggle={toggle}>Electricity Price</ModalHeader>
        <ModalBody>
          <VictoryChart
            theme={chartTheme}
            domainPadding={22}
            width={450}
            height={200}
            padding={{ top: 10, bottom: 30, left: 20, right: 20 }}
          >
            <VictoryBar
              data={data.allPrices.map((item) => ({
                x: item.DateTime,
                y: item.Price,
              }))}
              barRatio={0.5}
              style={{
                data: {
                  fill: ({ datum }) =>
                    datum.y >= data.dayAverage ? "#FF0046" : "rgb(0,255,121)",
                  fillOpacity: 0.9,
                  strokeWidth: 1,
                  stroke: ({ datum }) =>
                    Date.parse(datum.x) === new Date().setMinutes(0, 0, 0)
                      ? "rgb(255,255,255,0.9)"
                      : "none",
                },
              }}
            />
            <VictoryAxis
              dependentAxis
              style={{
                ticks: { fill: "transparent", size: 5 },
                tickLabels: { fontSize: 7 },
              }}
            />
            <VictoryAxis
              style={{
                ticks: { fill: "transparent", size: 5 },
                tickLabels: { fontSize: 7 },
              }}
              tickFormat={(t) => {
                const dt = new Date(Date.parse(t));
                if (dt.getMinutes() === 0) {
                  return dt.getHours();
                } else {
                  return "";
                }
              }}
            />
            <VictoryLine
              y={() => data.dayAverage}
              style={{
                data: {
                  strokeWidth: 0.5,
                  stroke: "rgb(0,255,121)",
                  strokeDasharray: "2, 2",
                },
              }}
            />
          </VictoryChart>
        </ModalBody>
      </Modal>
    </div>
  );

  return (
    <div id="elPrice" onClick={toggle}>
      {contents}
    </div>
  );
}

ElectricityPrice.displayName = "ElectricityPrice";
