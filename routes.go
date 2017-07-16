package main

import (
	"github.com/labstack/echo"
	"net/http"
	"encoding/json"
)

func getAlerts(c echo.Context) error {
	email := c.Param("email")
	var alerts []Alert
	db.Table("alerts").Where("email = ?", email).Find(&alerts)

	res, err := json.Marshal(alerts)
	if (err != nil) {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, string(res))
}

func getNotifications(c echo.Context) error {
	email := c.Param("email")
	var notifications []Notification
	db.Table("notifications").Where("email = ?", email).Find(&notifications)

	res, err := json.Marshal(notifications)
	if (err != nil) {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, string(res))
}

func countNotifications(c echo.Context) error {
	var count int64
	db.Table("notifications").Count(&count)
	return c.JSON(http.StatusOK, count)
}

func deleteAlert(c echo.Context) error {
	alert := new(Alert)
	if err := c.Bind(alert); err != nil {
		return err
	}
	db.Delete(&alert)

	return c.JSON(http.StatusOK, alert)
}


func deleteNotifications(c echo.Context) error {
	alert := new(Alert)
	if err := c.Bind(alert); err != nil {
		return err
	}
	db.Delete(&alert)

	return c.JSON(http.StatusOK, alert)
}

func addAlert(c echo.Context) error {
	alert := new(Alert)
	if err := c.Bind(alert); err != nil {
		return err
	}
	db.Create(&alert)

	return c.JSON(http.StatusCreated, alert)
}



