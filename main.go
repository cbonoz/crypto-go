package main

import (
	"fmt"
	"github.com/jasonlvhit/gocron"
	"net/http"
	"github.com/labstack/echo"
	_ "github.com/lib/pq"
	"github.com/buger/jsonparser"
	"io/ioutil"
	sms "stathat.com/c/amzses"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/op/go-logging"
	"time"
)

type Alert struct {
	gorm.Model
	email          string `json:"name"`
	coin           string `json:"coin"` // currency symbol
	timeDelta      string `json:"time_delta"`
	thresholdDelta float64 `json:"threshold_delta"`
	notes          string `json:"notes"`
	active         bool `json:"active"`
}

type Notification struct {
	gorm.Model
	alertId        uint `json:"alert_id"`
	email          string `json:"name"`
	coin           string `json:"coin"`
	currentDelta   float64 `json:"current_delta"`
	thresholdDelta float64 `json:"threshold_delta"`
}

var db gorm.DB
var log = logging.MustGetLogger("crypto")

const appName = "CryptoAlarms"
const emailDisplayName = "CryptoAlarms Notifications"
const domain = "https=//www.cryptoalarms.com/"
const adminEmail = "chris@blackshoalgroup.com"
const SIX_HOURS_MS = 1000 * 60 * 60 * 6


func insertNotification(n Notification) {
	_, err := db.Raw("insert into Notifications(id, alertId, email, coin, current_delta, threshold_delta, Created_at)" +
		" values($1, $2, $3, $4, $5, $6, $7)",
		n.ID, n.alertId, n.email, n.coin, n.currentDelta, n.thresholdDelta, n.CreatedAt).Rows()
	if (err != nil) {
		log.Error("error inserting notification: $1", err.Error())
	}
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

func noRecentViolations(email string, coin string) bool {
	// Retrieve the latest alert for the user for this particular coin (if present).
	rows, err := db.Raw("select * from Notifications where Created_at = " +
		"(select max(Created_at) from Notifications where email = $1 and coin = $2) " +
		"and email = $1 and coin = $2", email, coin).Rows()
	if (err != nil) {
		return true // no rows.
	}

	rows.Next()
	var notification Notification
	rows.Scan(&notification)
	// Return if the last notification was Created more than 6 hours ago.
	diff := time.Now().Sub(notification.CreatedAt)
	return diff.Hours() >= 6
}

func runCoinTask() {
	fmt.Println("runCoinTask")

	rows, err := db.Raw("select * from alerts where active = true").Rows()

	if (err != nil) {
		log.Error(err)
	}

	defer rows.Close()

	log.Debug("Found $1 active alerts", rows.Next())
	var coinDeltas = make(map[string][]byte)

	var notificationMap = make(map[string][]Notification)
	for rows.Next() {
		var alert *Alert
		err := rows.Scan(&alert)
		if err != nil {
			log.Error(err.Error())
		}

		var coinData []byte

		if val, ok := coinDeltas[alert.coin]; ok {
			coinData = val
		} else {
			// Coin not present in map, retrieve from api.
			res, err := http.Get("https://api.coinmarketcap.com/v1/ticker/")
			if (err != nil) {
				log.Error(err.Error())
			}

			body, err := ioutil.ReadAll(res.Body)

			coinDeltas[alert.coin] = body
			coinData = body
			// Close the response body after usage is complete.
			res.Body.Close()
		}

		// Parse the coin data for the change.
		var change float64
		if value, err := jsonparser.GetFloat(coinData, alert.timeDelta); err == nil {
			change = value
		}

		if (isViolation(change, alert.thresholdDelta) && noRecentViolations(alert.email, alert.coin)) {
			notification := Notification{
				alertId: alert.ID, email: alert.email, coin: alert.coin,
				currentDelta: change, thresholdDelta: alert.thresholdDelta,
			}

			insertNotification(notification)

			//if _, ok := notificationMap[alert.email]; !ok {
			//	notificationMap[alert.email] = []
			//}
			// Append notification to list.
			notificationMap[alert.email] = append(notificationMap[alert.email], notification)
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

func main() {

	e := echo.New()

	// Sample hello world route (for testing).
	e.GET("/hello", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	// Routes for manipulating alerts.
	e.PUT("/alerts/:email", addAlert)
	e.GET("/alerts/:email", getAlerts)
	e.DELETE("/alerts/:id", deleteAlert)

	// Routes for manipulating notifications generated by alerts.
	e.GET("/notifications/:email", getNotifications)
	//e.PUT("/notifications/:email", addNotification)

	var err error
	db, err := gorm.Open("postgres", "host=localhost user=cbono dbname=crypto sslmode=disable password=cbono")
	defer db.Close()
	if err != nil {
		log.Error(err.Error())
	}

	db.AutoMigrate(&Alert{}, &Notification{})
	db.Model(&Alert{}).AddIndex("idx_email", "email")
	db.Model(&Notification{}).AddIndex("idx_email", "email")
	db.Model(&Notification{}).AddForeignKey("alert_id", "alerts(id)", "RESTRICT", "RESTRICT")

	// Start the web server.
	port := ":9006"
	fmt.Println("Started server on port $1", port)
	e.Logger.Error(e.Start(port))

	runCoinTask()
	// TODO: readd task schedule after testing.
	//scheduleTask()
}



