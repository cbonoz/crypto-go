package main

import (
	"fmt"
	"github.com/jasonlvhit/gocron"
	"net/http"
	"github.com/labstack/echo"
	"io/ioutil"
	//sms "stathat.com/c/amzses"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/op/go-logging"
	"time"
	"github.com/labstack/echo/middleware"
	"strconv"
	"encoding/json"
	"strings"
)

type UserEmail struct {
	Email string `json:"email" form:"email" query:"email"`
}

type Alert struct {
	gorm.Model
	Name           string `json:"name"`
	Email          string `json:"email"`
	Coin           string `json:"coin"`
	ThresholdDelta float64 `json:"threshold_delta"`
	TimeDelta      string `json:"time_delta"`
	Active         bool `json:"active"`
}

type Notification struct {
	gorm.Model
	AlertId        uint
	Email          string
	Coin           string
	CurrentDelta   float64
	ThresholdDelta float64
	TimeDelta      string
	LastUpdated    int64
}

type CoinInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	Rank         string `json:"rank"`
	PriceUSD     string `json:"price_usd"`
	PriceBTC     string `json:"price_btc"`
	Volume24     string `json:"24h_volume_usd"`
	MarketCapUSD string `json:"market_cap_usd"`
	Supply       string `json:"available_supply"`
	TotalSupply  string `json:"total_supply"`
	Change1h     string `json:"percent_change_1h"`
	Change24h    string `json:"percent_change_24h"`
	Change7d     string `json:"percent_change_7d"`
	LastUpdated  string `json:"last_updated"`
}

var db *gorm.DB
var log = logging.MustGetLogger("crypto")

const MIN_HOUR_EMAIL_INTERVAL = 12
const COIN_API = "https://api.Coinmarketcap.com/v1/ticker/";

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func unixMilli(t time.Time) int64 {
	return t.Round(time.Millisecond).UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func insertNotification(n Notification) {
	log.Debugf("Inserting notification: %s", n)
	db.Create(&n)
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}


func sendNotificationsToUser(email string, alertMap map[string]Notification) string {
	const adminEmail = "chris@blackshoalgroup.com"
	const emailDisplayName = "CryptoAlarms Notifications"

	var coinsArr []string

	var alertNames []string
	var ns []Notification
	for alertName, notification := range alertMap {
		coinsArr = append(coinsArr, notification.Coin)
		ns = append(ns, notification)
		alertNames = append(alertNames, alertName)
	}
	coinString := strings.Join(coinsArr, ", ")
	coinString = coinString[0:min(len(coinString), 20)] // cap the string length at 20.

	var subject = fmt.Sprintf("[%s] %s passed change threshold.", emailDisplayName, coinString)
	//var body = createEmailBodyFromNotifications(alertNames, ns)

	// func SendMailHTML(from, to, subject, bodyText, bodyHTML string) (string, error)

	//res, err := sms.SendMailHTML(adminEmail, email, subject, "", body)
	//if (err != nil) {
	//	log.Error(email, err.Error())
	//}

	return subject
}

func isViolation(change float64, threshold float64) bool {
	return (threshold < 0 && change < threshold) || (threshold > 0 && change > threshold)
}

func noRecentViolations(email string, coin string) bool {
	// Retrieve the latest alert for the user for this particular Coin (if present).
	var notification Notification
	var err error
	err = db.Table("notifications").Where("coin = ? and email = ?", coin, email).Order("created_at desc").First(&notification).Error

	// Return false if the last notification was Created within the minimum interval.
	if (err != nil) {
		log.Debugf("First notification for (%s, %s)", coin, email)
		log.Error(err)
		return true
	}

	diff := time.Now().Sub(notification.CreatedAt)
	noRecentViolation := diff.Hours() >= MIN_HOUR_EMAIL_INTERVAL
	log.Debugf("Violation for coin %s, received notification within %d hours ago (hours ago: %d) - noRecentViolation(%s)",
		coin, MIN_HOUR_EMAIL_INTERVAL, diff.Hours(), noRecentViolation)

	return noRecentViolation
}

func printNotificationMap(x map[string]map[string]Notification) {
	b, err := json.MarshalIndent(x, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Print(string(b))
}

func getCurrencyPrices() map[string]CoinInfo {
	res, err := http.Get(COIN_API)
	if (err != nil) {
		log.Error(err.Error())
	}

	body, err := ioutil.ReadAll(res.Body)
	if (err != nil) {
		fmt.Println(err.Error())
	}
	var coinInfos []CoinInfo
	err = json.Unmarshal(body, &coinInfos)
	if (err != nil) {
		fmt.Println("error unmarshalling coin api response", err.Error())
	}

	log.Debug("found coininfo for ", len(coinInfos), " coins")

	var CoinDeltas = make(map[string]CoinInfo)
	for _, coinInfo := range coinInfos {
		coinSymbol := strings.ToUpper(coinInfo.Symbol)
		CoinDeltas[coinSymbol] = coinInfo
	}
	return CoinDeltas
}

func runCoinTask() {
	log.Debugf("runCoinTask: %s" + time.Now().String())
	var alerts []Alert

	db.Table("alerts").Where("active = true").Find(&alerts)

	numAlerts := len(alerts)
	log.Debug("Found active alerts: ", numAlerts)

	if (numAlerts == 0) {
		log.Debugf("No active alerts, returning from runCoinTask")
		return
	}

	CoinDeltas := getCurrencyPrices()

	var notificationMap = make(map[string]map[string]Notification)

	for _, alert := range alerts {
		// element is the element from someSlice for where we are

		coinSymbol := strings.ToUpper(alert.Coin)
		coinInfo, ok := CoinDeltas[coinSymbol]
		if !ok {
			log.Error("Could not find coin ", coinSymbol, " in api response")
		}


		// Parse the Coin data for the change.
		var change float64
		var err error

		switch alert.TimeDelta {
		case "7d":
			change, err = strconv.ParseFloat(coinInfo.Change7d, 64)
		case "1h":
			change, err = strconv.ParseFloat(coinInfo.Change1h, 64)
		case "24h":
			change, err = strconv.ParseFloat(coinInfo.Change24h, 64)
		default:
			log.Error("Unexpected alert.TimeDelta for alert ID($1): $2", alert.ID, alert.TimeDelta)
			change, err = strconv.ParseFloat(coinInfo.Change1h, 64)
		}

		if (err != nil) {
			log.Error(err.Error())
		}

		violation := isViolation(change, alert.ThresholdDelta)
		log.Debugf("CoinInfo %s: %s", alert.Coin, coinInfo)
		log.Debugf("Violation (%f, %f) - %s", change, alert.ThresholdDelta, violation)

		if (violation && noRecentViolations(alert.Email, alert.Coin)) {

			lastUpdated, err := strconv.ParseInt(coinInfo.LastUpdated, 10, 64)

			if (err != nil) {
				log.Error(err.Error())
				lastUpdated = makeTimestamp()
			}

			notification := Notification{
				AlertId: alert.ID, Email: alert.Email, Coin: alert.Coin, TimeDelta: alert.TimeDelta,
				CurrentDelta: change, ThresholdDelta: alert.ThresholdDelta, LastUpdated: lastUpdated,
			}

			insertNotification(notification)

			//if _, ok := notificationMap[alert.email]; !ok {
			//	notificationMap[alert.email] = []
			//}
			// Ensure that the map is initialized for the current user.
			_, ok := notificationMap[alert.Email]
			if !ok {
				notificationMap[alert.Email] = make(map[string]Notification)
			}
			// Append notification to alert map.
			notificationMap[alert.Email][alert.Name] = notification
		} // else no violation, continue.
	} // end row (alert config) iteration.

	log.Debug("done scanning alert table")
	log.Debugf("generating the following notifications:")
	printNotificationMap(notificationMap)

	// Send out the aggregated coin notification emails to user recipients.
	for email, alertMap := range notificationMap {
		fmt.Printf("key[%s] value[%v]\n", email, alertMap)
		res := sendNotificationsToUser(email, alertMap)
		log.Debug(email, res)
	}
}

func checkTables() {
	log.Debug("alerts table:" + strconv.FormatBool(db.HasTable("alerts")))
	log.Debug("notifications table:" + strconv.FormatBool(db.HasTable("notifications")))
}

func main() {

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// CORS default
	// Allows requests from any origin wth GET, HEAD, PUT, POST or DELETE method.
	// e.Use(middleware.CORS())

	// CORS restricted
	// Allows requests from particular web origins.
	// with GET, PUT, POST or DELETE method.
	// TODO: ONLY ALLOW REQUESTS THAT ORIGINATE FROM THE WEBSITE (security risk).
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"https://cryptoalarms.com", "https://www.cryptoalarms.com"},
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
	}))

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
	e.POST("/notifications/delete", deleteNotifications)
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
	db.Model(&Notification{}).AddIndex("notfication_idx_email", "email")
	db.Model(&Notification{}).AddForeignKey("alert_id", "alerts(ID)", "RESTRICT", "RESTRICT")

	// runCoinTask() // runs the coin check task once.
	var interval uint64
	interval = 30
	s := gocron.NewScheduler()
	s.Every(interval).Minutes().Do(runCoinTask)
	log.Debugf("scheduled alert task for every %d minutes", interval)
	s.Start()

	// Start the web server.
	port := ":8443"
	fmt.Println("Started server on port $1", port)
	e.Logger.Error(e.Start(port))
}



