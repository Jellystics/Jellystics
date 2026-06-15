const express = require("express");
const db = require("../db");
const jwt = require("jsonwebtoken");
const configClass = require("../classes/config");
const packageJson = require("../../package.json");
const API = require("../classes/api-loader");
const { axios } = require("../classes/axios");

const JWT_SECRET = process.env.JWT_SECRET;
if (JWT_SECRET === undefined) {
  console.log("JWT Secret cannot be undefined");
  process.exit(1);
}

const router = express.Router();

router.get("/isConfigured", async (req, res) => {
  try {
    const config = await new configClass().getConfig();
    res.json({
      state: config.state,
      version: packageJson.version,
      jellyfinUrl: config.JF_HOST || null,
    });
  } catch (error) {
    console.log(error);
    res.sendStatus(500);
  }
});

router.post("/check-server", async (req, res) => {
  try {
    const { url } = req.body;
    if (!url) return res.status(400).json({ error: "URL required" });

    const cleanUrl = url.replace(/\/$/, "");
    const response = await axios.get(`${cleanUrl}/System/Info/Public`, { timeout: 6000 });
    res.json({
      serverName: response.data.ServerName || "Jellyfin",
      version: response.data.Version,
      url: cleanUrl,
    });
  } catch {
    res.status(503).json({ error: "Server unreachable" });
  }
});

router.post("/jellyfin-login", async (req, res) => {
  try {
    const { username, password } = req.body;
    if (!username || !password) {
      return res.status(400).json({ error: "Username and password are required" });
    }

    const config = await new configClass().getConfig();
    const jfHost = config.JF_HOST;
    if (!jfHost) {
      return res.status(503).json({ error: "Jellyfin not configured" });
    }

    const authHeader =
      'MediaBrowser Client="Jellystics", Device="Jellystics Server", DeviceId="jellystics-server", Version="' +
      packageJson.version +
      '"';

    let response;
    try {
      response = await axios.post(
        `${jfHost}/Users/AuthenticateByName`,
        { Username: username, Pw: password },
        { headers: { Authorization: authHeader, "Content-Type": "application/json" }, timeout: 8000 }
      );
    } catch (err) {
      if (err.response?.status === 401) return res.sendStatus(401);
      // Timeout or network error → Jellyfin unreachable
      return res.status(503).json({ error: "Jellyfin unreachable" });
    }

    const jellyfinUser = response.data;

    if (!jellyfinUser.User?.Policy?.IsAdministrator) {
      return res.status(403).json({ error: "Administrator access required" });
    }

    const user = {
      id: jellyfinUser.User.Id,
      username: jellyfinUser.User.Name,
      jellyfinToken: jellyfinUser.AccessToken,
    };

    jwt.sign({ user }, JWT_SECRET, (err, token) => {
      if (err) { console.log(err); return res.sendStatus(500); }
      res.json({ token });
    });
  } catch (error) {
    console.log(error);
    res.sendStatus(500);
  }
});

router.post("/configSetup", async (req, res) => {
  try {
    const { JF_HOST, JF_API_KEY } = req.body;

    if (!JF_HOST || !JF_API_KEY) {
      return res.status(400).send("JF_HOST and JF_API_KEY are required");
    }

    const validation = await API.validateSettings(JF_HOST, JF_API_KEY);
    if (validation.isValid === false) {
      res.status(validation.status);
      return res.send(validation);
    }

    const { rows: getConfig } = await db.query('SELECT * FROM app_config where "ID"=1');

    let query = 'UPDATE app_config SET "JF_HOST"=$1, "JF_API_KEY"=$2 where "ID"=1';
    if (getConfig.length === 0) {
      query = 'INSERT INTO app_config ("ID","JF_HOST","JF_API_KEY") VALUES (1,$1,$2)';
    }

    const { rows } = await db.query(query, [validation.cleanedUrl, JF_API_KEY]);

    const systemInfo = await API.systemInfo();
    if (systemInfo) {
      const settingsjson = await db.query('SELECT settings FROM app_config where "ID"=1').then((r) => r.rows);
      if (settingsjson.length > 0) {
        const settings = settingsjson[0].settings || {};
        settings.Tasks = systemInfo?.Id || null;
        await db.query('UPDATE app_config SET settings=$1 where "ID"=1', [settings]);
      }
    }

    res.send(rows);
  } catch (error) {
    console.log(error);
    res.sendStatus(500);
  }
});

router.use((req, res) => {
  res.status(404).send({ error: "Not Found" });
});

module.exports = router;
