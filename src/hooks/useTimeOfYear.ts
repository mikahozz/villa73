export const getTimeOfYear = (date: Date = new Date()) => {
  const month = date.getMonth() + 1; // getMonth() is zero-based
  if (month >= 3 && month <= 5) {
    return "spring";
  } else if (month >= 6 && month <= 8) {
    return "summer";
  } else if (month >= 9 && month <= 11) {
    return "autumn";
  } else {
    return "winter";
  }
};
