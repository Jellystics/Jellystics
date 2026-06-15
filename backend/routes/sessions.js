const express = require("express");
const API = require("../classes/api-loader");

const router = express.Router();

router.get("/current", async (req, res) => {
  try {
    const sessions = await API.getSessions();
    res.send(sessions ?? []);
  } catch (error) {
    res.status(503);
    res.send(error);
  }
});

router.use((req, res) => {
  res.status(404).send({ error: "Not Found" });
});

module.exports = router;
