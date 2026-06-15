// Query returns columns in this order (with rowid):
// [0] rowid, [1] DateCreated, [2] UserId, [3] ItemId, [4] ItemType, [5] ItemName,
// [6] PlaybackMethod, [7] ClientName, [8] DeviceName, [9] PlayDuration
const columnsPlaybackReporting = [
  "rowid",
  "DateCreated",
  "UserId",
  "ItemId",
  "ItemType",
  "ItemName",
  "PlaybackMethod",
  "ClientName",
  "DeviceName",
  "PlayDuration",
];

const mappingPlaybackReporting = (item) => {
  let duration = item[9];
  if (duration === null || duration === undefined || duration < 0) {
    duration = 0;
  }
  return {
    rowid: String(item[0]),
    DateCreated: item[1],
    UserId: item[2],
    ItemId: item[3],
    ItemType: item[4],
    ItemName: item[5],
    PlaybackMethod: item[6],
    ClientName: item[7],
    DeviceName: item[8],
    PlayDuration: duration,
  };
};

module.exports = {
  columnsPlaybackReporting,
  mappingPlaybackReporting,
};
