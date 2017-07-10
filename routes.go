package main

import (
	"github.com/labstack/echo"
	"time"
	"github.com/rs/xid"
	"net/http"
)

func deleteAlert(c echo.Context) error {
	id := c.Param("id")
	_, err := db.Query("delete * from CoinAlerts where id=$1", id)
	if (err) {
		return c.String(http.StatusBadRequest, err)
	}

	return c.String(http.StatusOK, id)
}

func getAlerts(c echo.Context) error {
	email := c.Param("email")
	rows, err := db.Query("SELECT name FROM CoinAlerts WHERE email = $email", email)
	defer rows.Close()

	if (err) {
		return c.String(http.StatusInternalServerError, err)
	}

	return c.String(http.StatusOK, rows)
}

func getNotifications(c echo.Context) error {
	email := c.Param("email")
	rows, err := db.Query("SELECT name FROM Notifications WHERE email = $email", email)

	if (err) {
		return c.String(http.StatusInternalServerError, err)
	}

	return c.String(http.StatusOK, rows)
}

func addAlert(c echo.Context) error {
	email := c.Param("email")
	coin := c.Param("coin")
	thresholdDelta := c.Param("threshold_delta")
	timeDelta := c.Param("time_delta")
	notes := c.Param("notes")
	active := c.Param("active")
	createdAt := time.Now().UTC()
	guid := xid.New()
	_, err := db.Query("insert into Alerts(id, email, coin, threshold_delta, time_delta, created_at) " +
		"values($1, $2, $3, $4, $5, $6, $7, $8))", guid, email, coin, thresholdDelta, timeDelta, notes, active, createdAt)

	if (err) {
		return c.String(http.StatusInternalServerError, err)
	}

	// TODO: replace with standardized response for successful creation of notification.
	res := "added alert"
	return c.String(http.StatusOK, res)
}

func addNotification(c echo.Context) error {
	alertId := c.Param("alertId")
	email := c.Param("email")
	coin := c.Param("coin")
	thresholdDelta := c.Param("threshold_delta")
	currentDelta := c.Param("current_delta")
	guid := xid.New()

	createdAt := time.Now().UTC()
	_, err := db.Query("insert into Notifications(id, alertId, email, coin, current_delta, threshold_delta, created_at)" +
		" values($1, $2, $3, $4, $5, $6, $7)", guid, alertId, email, coin, currentDelta, thresholdDelta, createdAt)

	if (err) {
		return c.String(http.StatusInternalServerError, err)
	}

	// TODO: replace with standardized response for successful creation of notification.
	res := "added notification"
	return c.String(http.StatusOK, res)
}

//func save(c echo.Context) error {
//	// Get name and email
//	name := c.FormValue("email")
//	email := c.FormValue("email")
//	return c.String(http.StatusOK, "name:" + name + ", email:" + email)
//}



