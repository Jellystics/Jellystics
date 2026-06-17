package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// UtilsHandler handles miscellaneous utility endpoints.
type UtilsHandler struct{}

func NewUtilsHandler() *UtilsHandler { return &UtilsHandler{} }

// isPublicIPv4 returns true if the string is a valid IPv4 address that is not
// in a private/loopback range (10.x, 172.16-31.x, 192.168.x, 127.x).
func isPublicIPv4(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	ip = ip.To4()
	if ip == nil {
		return false
	}
	private := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8"}
	for _, cidr := range private {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return false
		}
	}
	return true
}

// POST /utils/geolocateIp
func (h *UtilsHandler) GeolocateIP(c *gin.Context) {
	accountID := os.Getenv("JS_GEOLITE_ACCOUNT_ID")
	licenseKey := os.Getenv("JS_GEOLITE_LICENSE_KEY")

	if accountID == "" || licenseKey == "" {
		c.JSON(http.StatusNotImplemented, "GeoLite information missing!")
		return
	}

	var body struct {
		IPAddress string `json:"ipAddress"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || !isPublicIPv4(body.IPAddress) {
		c.JSON(http.StatusBadRequest, "Invalid IP address sent!")
		return
	}

	url := fmt.Sprintf("https://geolite.info/geoip/v2.1/city/%s", body.IPAddress)
	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.SetBasicAuth(accountID, licenseKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		c.Data(resp.StatusCode, "application/json", data)
		return
	}
	c.JSON(resp.StatusCode, result)
}
