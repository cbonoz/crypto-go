package main

import (
	"github.com/labstack/echo"
	"net/http"
)

type (

	alertHandler struct {
		db map[string]*Alert
	}

	notificationHandler struct {
		db map[string]*Notification
	}
)

func (h *alertHandler) createAlert(c echo.Context) error {
	u := new(Alert)
	if err := c.Bind(u); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, u)
}

func (h *alertHandler) getAlerts(c echo.Context) error {
	email := c.Param("email")
	user := h.db[email]
	if user == nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	return c.JSON(http.StatusOK, user)
}

func (h *notificationHandler) createNotification(c echo.Context) error {
	u := new(Notification)
	if err := c.Bind(u); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, u)
}

func (h *notificationHandler) getNotifications(c echo.Context) error {
	email := c.Param("email")
	user := h.db[email]
	if user == nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	return c.JSON(http.StatusOK, user)
}


