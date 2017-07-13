package main

import (
	"github.com/labstack/echo"
	"net/http"
	"encoding/json"
	"database/sql"
	"strconv"
)

func getArrayStringFromRows(rows *sql.Rows) string {
	var res = "["

	for rows.Next() {
		var alert Alert
		rows.Scan(&alert)
		out, err := json.Marshal(alert)
		if err != nil {
			log.Error(err)
		}
		res += string(out) + ","
	}

	sz := len(res)
	if (sz > 1) {
		// strip off last comma.
		res = res[:sz - 1]
	}
	res += "]"
	return res
}

func deleteAlert(c echo.Context) error {
	id := c.Param("id")
	rows, err := db.Raw("delete * from alerts where id=$1", id).Rows()
	defer rows.Close()
	if (err != nil) {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.String(http.StatusOK, id)
}

func getAlerts(c echo.Context) error {
	email := c.Param("email")
	rows, err := db.Raw("SELECT * FROM alerts WHERE email = $1", email).Rows()
	defer rows.Close()

	if (err != nil) {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, getArrayStringFromRows(rows))
}

func getNotifications(c echo.Context) error {
	email := c.Param("email")
	rows, err := db.Raw("SELECT name FROM notifications WHERE email = $1", email).Rows()
	defer rows.Close()

	if (err != nil) {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, getArrayStringFromRows(rows))
}

func countNotifications(c echo.Context) error {
	var count int64
	db.Table("notifications").Count(&count)
	return c.String(http.StatusOK, string(count))
}

func addAlert(c echo.Context) error {
	name := c.Param("name")
	email := c.Param("email")
	coin := c.Param("coin")
	notes := c.Param("notes")
	timeDelta := c.Param("time_delta")
	thresholdString := c.Param("threshold_delta")
	activeString := c.Param("active")

	thresholdDelta, err  := strconv.ParseFloat(thresholdString, 64)
	if (err != nil) {
		return c.String(http.StatusBadRequest, "threshold must be a float")
	}

	active, err  := strconv.ParseBool(activeString)
	if (err != nil) {
		return c.String(http.StatusBadRequest, "active must be a true or false value")
	}
	alert := Alert{Name: name, Email: email, Coin: coin, ThresholdDelta: thresholdDelta, TimeDelta: timeDelta,
		Notes: notes, Active: active}

	db.Create(&alert)

	// TODO: replace with standardized response for successful creation of notification.
	res := "added alert"
	return c.String(http.StatusOK, res)
}

//func save(c echo.Context) error {
//	// Get name and email
//	name := c.FormValue("email")
//	email := c.FormValue("email")
//	return c.String(http.StatusOK, "name:" + name + ", email:" + email)
//}



