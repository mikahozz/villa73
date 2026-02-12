import { useState, useEffect } from "react";
import moment from "moment";
import _ from "lodash";
import { Modal, ModalHeader, ModalBody } from "reactstrap";

interface IndoorData {
  temperature: number;
  humidity: number;
  battery: number;
  time: string;
}

export function Balcony() {
  const [modal, setModal] = useState(false);
  const [indoordata, setIndoordata] = useState<IndoorData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    populateIndoorData();
    // Refresh data every 10 minutes
    const intervalId = setInterval(populateIndoorData, 10 * 60 * 1000);

    // Cleanup on unmount
    return () => clearInterval(intervalId);
  }, []);

  const populateIndoorData = async () => {
    try {
      const response = await fetch("/api/indoor/Shelly");
      const data = await response.json();
      setIndoordata(data);
      setLoading(false);
    } catch (error) {
      console.error("Failed to fetch indoor data:", error);
    }
  };

  const oudatedTimeMs = 1000 * 60 * 60 * 12;
  const isOutdated = indoordata
    ? new Date().getTime() - new Date(indoordata.time).getTime() > oudatedTimeMs
    : true;

  const renderUpdatedClasses = (date: number) => {
    const diff = Math.abs(new Date().getTime() - date);
    let cssClass = "dateUpdated";
    if (diff / oudatedTimeMs > 1) {
      cssClass += " outdated";
    }
    return cssClass;
  };

  const toggle = () => {
    setModal(!modal);
  };

  const contents = loading ? (
    <div>
      <p className="balconyTemp">
        <em>...</em>
      </p>
    </div>
  ) : (
    <div>
      <p className="balconyTemp">{_.round(indoordata?.temperature || 0, 1)}°</p>
      <Modal isOpen={modal} toggle={toggle}>
        <ModalHeader toggle={toggle}>Indoor temperature</ModalHeader>
        <ModalBody>
          <p>
            <span className="balconyTemp">
              {_.round(indoordata?.temperature || 0, 1)}°
            </span>
            <br />
            Humidity: {_.round(indoordata?.humidity || 0, 1)}%<br />
            Battery: {indoordata?.battery}%<br />
            Updated:{" "}
            <span
              className={renderUpdatedClasses(
                Date.parse(indoordata?.time || "")
              )}
            >
              {moment(indoordata?.time).format("ddd HH:mm")}
            </span>
          </p>
        </ModalBody>
      </Modal>
      {isOutdated && <p className="alert">!</p>}
    </div>
  );

  return (
    <div id="balcony" onClick={toggle}>
      {contents}
    </div>
  );
}

Balcony.displayName = "Balcony";
