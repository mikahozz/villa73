import { useState } from "react";
import useWeatherNow from "../hooks/useWeatherNow";
import { Modal, ModalHeader, ModalBody } from "reactstrap";

export function WeatherNow() {
  const [modal, setModal] = useState(false);
  const { data: weatherdata, isPending, isError, error } = useWeatherNow();

  const toggle = () => {
    setModal(!modal);
  };

  const content = weatherdata
    ? `${weatherdata[weatherdata.length - 1].temperature}°`
    : isPending
    ? "..."
    : isError
    ? `!`
    : "-";
  const isOutdated = weatherdata
    ? new Date().getTime() -
        new Date(weatherdata[weatherdata.length - 1].datetime).getTime() >
      1000 * 60 * 60
    : true;

  return (
    <div id="weatherNow" onClick={toggle}>
      <p className="temperatureNow">{content}</p>
      <Modal isOpen={modal} toggle={toggle}>
        <ModalHeader toggle={toggle}>Weather now</ModalHeader>
        <ModalBody>
          <p>
            Updated:{" "}
            <span
              className={isOutdated ? "dateUpdated outdated" : "dateUpdated"}
            >
              {(weatherdata &&
                weatherdata.length &&
                weatherdata[weatherdata.length - 1].datetime) ??
                "-"}
            </span>
            <br />
            {error instanceof Error ? `Error:${error.message}` : ""}
          </p>
        </ModalBody>
      </Modal>
      {isOutdated && <p className="alert">!</p>}
    </div>
  );
}

WeatherNow.displayName = "WeatherNow";
