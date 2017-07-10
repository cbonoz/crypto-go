package main

import (
	"fmt"
	"github.com/jasonlvhit/gocron"
	"net/http"
	"github.com/labstack/echo"
	_ "github.com/lib/pq"
	"database/sql"
	"github.com/labstack/gommon/log"
	"time"
	"github.com/buger/jsonparser"
	"io/ioutil"
	"github.com/rs/xid"
	"container/list"
	sms "stathat.com/c/amzses"
)

type Alert struct {
	id             int64  `json:"id"`
	email          string `json:"name"`
	coin           string `json:"coin"` // currency symbol
	timeDelta      string `json:"time_delta"`
	thresholdDelta float32 `json:"threshold_delta"`
	createdAt      int64 `json:"createdAt"`
	notes          string `json:"notes"`
	active         bool `json:"active"`
}

type Notification struct {
	id             int64  `json:"id"`
	alertId        string `json:"alert_id"`
	email          string `json:"name"`
	coin           string `json:"coin"`
	currentDelta   float32 `json:"current_delta"`
	thresholdDelta float32 `json:"threshold_delta"`
	createdAt      int64 `json:"createdAt"`
}

var db *sql.DB

const appName = "CryptoAlarms"
const emailDisplayName = "CryptoAlarms Notifications"
const domain = "https=//www.cryptoalarms.com/"
const adminEmail = "chris@blackshoalgroup.com"
const SIX_HOURS_MS = 1000*60*60*6

func makeTimestamp()int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond))
}

func insertNotification(n Notification) {
	_, err := db.Query("insert into Notifications(id, alertId, email, coin, current_delta, threshold_delta, created_at)" +
		" values($1, $2, $3, $4, $5, $6, $7)",
		n.id, n.alertId, n.email, n.coin, n.currentDelta, n.thresholdDelta, n.createdAt)
	if (err) {
		log.ERROR("error inserting notification: $1", err)
	}
}

func sendNotificationsToUser(email string, ns []Notification) string {
	var body = createEmailBodyFromNotifications(ns)
	var subject = fmt.Sprintf("[%s] Coins passed change threshold.",
		emailDisplayName)

	// func SendMailHTML(from, to, subject, bodyText, bodyHTML string) (string, error)

	res, err := sms.SendMailHTML(adminEmail, email, subject, "", body)
	if (err) {
		log.ERROR(email, err)
	}

	return res
}

func isViolation(change float32, threshold float32) bool {
	return (threshold < 0 && change < threshold) || (threshold > 0 && change > threshold)
}

func noRecentViolations(email string, coin string) bool {
	// Retrieve the latest alert for the user for this particular coin (if present).
	rows, err := db.Query("select * from Notifications where created_at = " +
		"(select max(created_at) from Notifications where email = $1 and coin = $2) " +
		"and email = $1 and coin = $2", email, coin)
	if (err) {
		return true // no rows.
	}

	rows.Next()
	var notification Notification
	rows.Scan(&notification)
	// Return if the last notification was created more than 6 hours ago.
	return notification.createdAt > SIX_HOURS_MS
}

func runCoinTask() {
	fmt.Println("runCoinTask")

	rows, err := db.Query("select * from CoinAlerts where active = true")

	if (err) {
		log.ERROR(err)
	}

	defer rows.Close()

	log.DEBUG("Found $1 active alerts", rows.Next())
	var coinDeltas = make(map[string][]byte)

	var notificationMap = make(map[string][]Notification)
	for rows.Next() {
		var alert *Alert
		err := rows.Scan(&alert)
		if err != nil {
			log.Fatal(err)
		}

		var coinData []byte

		if val, ok := coinDeltas[alert.coin]; ok {
			coinData = val
		} else {
			// Coin not present in map, retrieve from api.
			res, err := http.Get("https://api.coinmarketcap.com/v1/ticker/")
			if (err) {
				log.Fatal(err)
			}

			body, err := ioutil.ReadAll(res.Body)

			coinDeltas[alert.coin] = body
			coinData = body
			// Close the response body after usage is complete.
			res.Body.Close()
		}

		// Parse the coin data for the change.
		var change float32
		if value, err := jsonparser.GetFloat(coinData, alert.timeDelta); err == nil {
			change = value
		}

		if (isViolation(change, alert.thresholdDelta) && noRecentViolations(alert.email, alert.coin)) {
			guid := xid.New()
			createdAt := makeTimestamp()
			notification := Notification{
				guid, alert.id, alert.email, alert.coin, change, alert.thresholdDelta, createdAt,
			}

			insertNotification(notification)

			if _, ok := notificationMap[alert.email]; !ok {
				notificationMap[alert.email] = list.New()
			}
			// Append notification to list.
			notificationMap[alert.email] = append(notificationMap[alert.email], notification)
		} // else no violation, continue.


	} // end row (alert config) iteration.

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	// Send out the aggregated emails.
	for k, v := range notificationMap {
		fmt.Printf("key[%s] value[%s]\n", k, v)
		res := sendNotificationsToUser(k, v)
		log.DEBUG(k, res)
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
	db, err = sql.Open("postgres", "user=pqgotest dbname=pqgotest sslmode=verify-full")
	if err != nil {
		log.ERROR(err)
	}

	// Start the web server.
	port := ":9006"
	fmt.Println("Started server on port $1", port)
	e.Logger.Error(e.Start(port))

	runCoinTask()
	// TODO: readd task schedule after testing.
	//scheduleTask()
}



