import { useEffect, useState } from "react";
import _ from "lodash";

export function Solar() {
  const [data, setData] = useState({ currentPower: 0 });
  useEffect(() => {
    const fetchData = async () => {
      await fetch("/api/electricity/current")
        .then((response) => {
          if (response.status === 200) {
            response
              .json()
              .then((json) => {
                console.log("Solar json: ", json);
                console.log("Current solar: " + json.powerw);
                setData({ currentPower: json.powerw });
              })
              .catch(() => {
                setData({ currentPower: 0 });
              });
          } else {
            setData({ currentPower: 0 });
          }
        })
        .catch((error) => {
          console.log(error);
          setData({ currentPower: 0 });
        });
    };
    const id = setInterval(() => {
      fetchData();
    }, 60 * 1000);
    fetchData();
    return () => clearInterval(id);
  }, []);

  return (
    <div
      className="solarBar"
      style={data.currentPower === 0 ? { opacity: 0.5 } : {}}
    >
      <div className="powerRow">
        <span
          className="powerW"
          style={
            data.currentPower / 4760 > 0.5
              ? { float: "left", color: "#000", textShadow: "0 0 5px #fff" }
              : {}
          }
        >
          {data.currentPower} W
        </span>
        <span
          className="powerNow"
          style={{ width: _.round((data.currentPower / 4760) * 100) + "%" }}
        />
      </div>
    </div>
  );
}
