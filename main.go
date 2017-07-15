package main

import (
	"fmt"
	"github.com/jasonlvhit/gocron"
	"net/http"
	"github.com/labstack/echo"
	"github.com/buger/jsonparser"
	"io/ioutil"
	sms "stathat.com/c/amzses"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/op/go-logging"
	"time"
	"github.com/labstack/echo/middleware"
	"strconv"
)

type Alert struct {
	gorm.Model
	Name           string `json:"name"`
	Email          string `json:"email"`
	Coin           string `json:"coin"`
	ThresholdDelta float64 `json:"threshold_delta"`
	TimeDelta      string `json:"time_delta"`
	Notes          string `json:"notes"`
	Active         bool `json:"active"`
}

type Notification struct {
	gorm.Model
	AlertId        uint
	Email          string
	Coin           string
	CurrentDelta   float64
	ThresholdDelta float64
}

var db *gorm.DB
var log = logging.MustGetLogger("crypto")

const appName = "CryptoAlarms"
const emailDisplayName = "CryptoAlarms Notifications"
const domain = "https=//www.cryptoalarms.com/"
const adminEmail = "chris@blackshoalgroup.com"
const MIN_HOUR_EMAIL_INTERVAL = 12

func insertNotification(n Notification) {
	db.Create(&n)
}

func sendNotificationsToUser(email string, ns []Notification) string {
	var body = createEmailBodyFromNotifications(ns)
	var subject = fmt.Sprintf("[%s] Coins passed change threshold.",
		emailDisplayName)

	// func SendMailHTML(from, to, subject, bodyText, bodyHTML string) (string, error)

	res, err := sms.SendMailHTML(adminEmail, email, subject, "", body)
	if (err != nil) {
		log.Error(email, err.Error())
	}

	return res
}

func isViolation(change float64, threshold float64) bool {
	return (threshold < 0 && change < threshold) || (threshold > 0 && change > threshold)
}

func noRecentViolations(email string, Coin string) bool {
	// Retrieve the latest alert for the user for this particular Coin (if present).
	rows, err := db.Raw("select * from Notifications where Created_at = " +
		"(select max(Created_at) from Notifications where email = $1 and Coin = $2) " +
		"and email = $1 and Coin = $2", email, Coin).Rows()
	if (err != nil) {
		return true // no rows.
	}

	rows.Next()
	var notification Notification
	rows.Scan(&notification)
	// Return false if the last notification was Created within the minimum interval.
	diff := time.Now().Sub(notification.CreatedAt)
	return diff.Hours() >= MIN_HOUR_EMAIL_INTERVAL
}

func runCoinTask() {
	fmt.Println("runCoinTask")
	var alerts []Alert

	db.Table("alerts").Where("active = true").Find(&alerts)

	log.Debug("Found $1 active alerts", len(alerts))
	var CoinDeltas = make(map[string][]byte)

	var notificationMap = make(map[string][]Notification)
	for _, alert := range alerts {
		// element is the element from someSlice for where we are

		var CoinData []byte

		if val, ok := CoinDeltas[alert.Coin]; ok {
			CoinData = val
		} else {
			// Coin not present in map, retrieve from api.
			res, err := http.Get("https://api.Coinmarketcap.com/v1/ticker/")
			if (err != nil) {
				log.Error(err.Error())
			}

			body, err := ioutil.ReadAll(res.Body)

			CoinDeltas[alert.Coin] = body
			CoinData = body
			// Close the response body after usage is complete.
			res.Body.Close()
		}

		// Parse the Coin data for the change.
		var change float64
		if value, err := jsonparser.GetFloat(CoinData, alert.TimeDelta); err == nil {
			change = value
		}

		if (isViolation(change, alert.ThresholdDelta) && noRecentViolations(alert.Email, alert.Coin)) {
			notification := Notification{
				AlertId: alert.ID, Email: alert.Email, Coin: alert.Coin,
				CurrentDelta: change, ThresholdDelta: alert.ThresholdDelta,
			}

			insertNotification(notification)

			//if _, ok := notificationMap[alert.email]; !ok {
			//	notificationMap[alert.email] = []
			//}
			// Append notification to list.
			notificationMap[alert.Email] = append(notificationMap[alert.Email], notification)
		} // else no violation, continue.


	} // end row (alert config) iteration.

	// Send out the aggregated emails.
	for k, v := range notificationMap {
		fmt.Printf("key[%s] value[%s]\n", k, v)
		res := sendNotificationsToUser(k, v)
		log.Debug(k, res)
	}
}

func scheduleTask() {
	// runCoinTask executes each 30 minutes.
	s := gocron.NewScheduler()
	s.Every(30).Minutes().Do(runCoinTask)
	<-s.Start()
}

func checkTables() {
	log.Debug("alerts table:" +  strconv.FormatBool(db.HasTable("alerts")))
	log.Debug("notifications table:" +  strconv.FormatBool(db.HasTable("notifications")))
}

func main() {

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// CORS default
	// Allows requests from any origin wth GET, HEAD, PUT, POST or DELETE method.
	// TODO: remove CORS - ONLY ALLOW REQUESTS THAT ORIGINATE FROM THE WEBSITE (security risk).
	e.Use(middleware.CORS())

	// CORS restricted
	// Allows requests from any `https://labstack.com` or `https://labstack.net` origin
	// wth GET, PUT, POST or DELETE method.
	//e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
	//	AllowOrigins: []string{"https://labstack.com", "https://labstack.net"},
	//	AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
	//}))

	// Sample hello world routes (for testing).
	e.GET("/hello", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "Hello, World!")
	})
	e.GET("/hello/:name", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "Hello, " + c.Param("name"))
	})

	// Routes for manipulating alerts.
	e.POST("/alerts", addAlert)
	e.POST("/alerts/delete", deleteAlert)
	e.GET("/alerts/:email", getAlerts)

	// Routes for manipulating notifications generated by alerts.
	e.GET("/notifications/:email", getNotifications)
	e.GET("/notifications/count", countNotifications)
	//e.PUT("/notifications/:email", addNotification) // Notifications are only added server-side.

	var err error
	// Create global db.
	db, err = gorm.Open("postgres", "host=localhost user=cbono dbname=crypto sslmode=disable password=cbono")
	defer db.Close()
	if err != nil {
		log.Error(err.Error())
	}
	checkTables()
	db.AutoMigrate(&Alert{}, &Notification{})
	log.Debug("tables migrated")
	// After migration.
	checkTables()

	db.Model(&Alert{}).AddIndex("alert_idx_email", "email")
	db.Model(&Notification{}).AddIndex("not_idx_email", "email")
	db.Model(&Notification{}).AddForeignKey("alert_id", "alerts(ID)", "RESTRICT", "RESTRICT")

	// Start the web server.
	port := ":9006"
	fmt.Println("Started server on port $1", port)
	e.Logger.Error(e.Start(port))

	runCoinTask()
	// TODO: readd task schedule after testing.
	//scheduleTask()
}



