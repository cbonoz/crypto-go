package main

import (
	"testing"
	"github.com/labstack/echo"
	"net/http/httptest"
	"strings"
	"github.com/stretchr/testify/assert"
	"net/http"
)

var (
	mockAlertDB = map[string]*Alert{"jon@labstack.com":
	&Alert{Name: "btc alert", Email:"jon@labstack.com",
		Coin: "BTC", ThresholdDelta:.7, TimeDelta:"7d", Notes:""},
	}
	mockNotificationDB = map[string]*Notification{"jon@labstack.com":
	&Notification{Email:"jon@labstack.com", Coin: "BTC", ThresholdDelta:.7, CurrentDelta:.8},
	}
	alertJson = `{"ID":0,"CreatedAt":"0001-01-01T00:00:00Z","UpdatedAt":"0001-01-01T00:00:00Z","DeletedAt":null,"name":"btc alert","email":"jon@labstack.com","coin":"BTC","threshold_delta":0.7,"time_delta":"7d","notes":"","active":false}`
	notificationJson = `{"ID":0,"CreatedAt":"0001-01-01T00:00:00Z","UpdatedAt":"0001-01-01T00:00:00Z","DeletedAt":null,"AlertId":0,"Email":"jon@labstack.com","Coin":"BTC","CurrentDelta":0.8,"ThresholdDelta":0.7}`
)

func TestCreateAlert(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(alertJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := &alertHandler{mockAlertDB}

	// Assertions
	if assert.NoError(t, h.createAlert(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, alertJson, rec.Body.String())
	}
}

func TestGetAlerts(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/users/:email")
	c.SetParamNames("email")
	c.SetParamValues("jon@labstack.com")
	h := &alertHandler{mockAlertDB}

	// Assertions
	if assert.NoError(t, h.getAlerts(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, alertJson, rec.Body.String())
	}
}

func TestCreateNotification(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(notificationJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := &notificationHandler{mockNotificationDB}

	// Assertions
	if assert.NoError(t, h.createNotification(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, notificationJson, rec.Body.String())
	}
}

func TestGetNotifications(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/users/:email")
	c.SetParamNames("email")
	c.SetParamValues("jon@labstack.com")
	h := &notificationHandler{mockNotificationDB}

	// Assertions
	if assert.NoError(t, h.getNotifications(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, notificationJson, rec.Body.String())
	}
}


