package main

import (
	"fmt"
	"github.com/jasonlvhit/gocron"
	"net/http"
	"github.com/labstack/echo"
	sms "stathat.com/c/amzses"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/op/go-logging"
	"time"
	"github.com/labstack/echo/middleware"
	"strconv"
	"encoding/json"
	"strings"
	"github.com/levigross/grequests"
	"os"
)

type UserEmail struct {
	Email string `json:"email" form:"email" query:"email"`
}

type Alert struct {
	gorm.Model
	Name           string `json:"name"`
	Email          string `json:"email"`
	CoinName       string `json:"coin_name"`
	CoinSymbol     string `json:"coin_symbol"`
	ThresholdDelta float64 `json:"threshold_delta"`
	TimeDelta      string `json:"time_delta"`
	Active         bool `json:"active"`
}

type Notification struct {
	gorm.Model
	AlertId        uint
	Email          string
	CoinName       string
	CoinSymbol     string
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

const MIN_HOUR_EMAIL_INTERVAL = 12.0
const COIN_API = "https://api.coinmarketcap.com/v1/ticker/";

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func unixMilli(t time.Time) int64 {
	return t.Round(time.Millisecond).UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func createCoinKey(coinSymbol string, coinName string) string {
	return strings.ToUpper(coinSymbol + "_" + coinName)
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

func sendNotificationsToUser(email string, notificationMap map[string]Notification) string {
	const adminEmail = "cryptoalarms@gmail.com"
	const emailDisplayName = "CryptoAlarms Notifications"

	var coinsArr []string

	var alertNames []string
	var ns []Notification
	for alertName, notification := range notificationMap {
		coinsArr = append(coinsArr, notification.CoinSymbol)
		ns = append(ns, notification)
		alertNames = append(alertNames, alertName)
	}
	coinString := strings.Join(coinsArr, ", ")
	coinString = coinString[0:min(len(coinString), 20)] // cap the string length at 20.

	var subject = fmt.Sprintf("[%s] %s passed change threshold.", emailDisplayName, coinString)
	var body = createEmailBodyFromNotifications(alertNames, ns)

	//func SendMailHTML(from, to, subject, bodyText, bodyHTML string) (string, error)

	res, err := sms.SendMailHTML(adminEmail, email, subject, "", body)
	if (err != nil) {
		log.Error(email, err.Error())
	}
	log.Debugf("Sending email for %s, result: %s", email, res)

	return subject
}

func isViolation(change float64, threshold float64) bool {
	return (threshold < 0 && change < threshold) || (threshold > 0 && change > threshold)
}

func noRecentViolations(email string, coinSymbol string, coinName string) bool {
	// Retrieve the latest alert for the user for this particular Coin (if present).
	var notification Notification
	var err error
	err = db.Table("notifications").Where("email = ? AND coin_symbol = ? AND coin_name = ?",
		email, coinSymbol, coinName).Order("created_at desc").First(&notification).Error

	// Return false if the last notification was Created within the minimum interval.
	if (err != nil) {
		log.Debugf("First notification for (%s, %s)", coinSymbol, email)
		log.Debugf("notification: %s", notification)
		log.Error(err)
		return true
	}

	diff := time.Now().Sub(notification.CreatedAt)
	noRecentViolation := diff.Hours() >= MIN_HOUR_EMAIL_INTERVAL
	log.Debugf("Violation for coin %s, received notification within %d hours ago (hours ago: %d) - noRecentViolation(%s)",
		coinSymbol, MIN_HOUR_EMAIL_INTERVAL, diff.Hours(), noRecentViolation)

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
	resp, err := grequests.Get(COIN_API, nil)
	if (err != nil) {
		log.Error(err.Error())
	}

	var coinInfos []CoinInfo
	err = json.Unmarshal(resp.Bytes(), &coinInfos)
	if (err != nil) {
		fmt.Println("error unmarshalling coin api response", err.Error())
	}

	log.Debug("found coininfo for ", len(coinInfos), " coins")

	var CoinDeltas = make(map[string]CoinInfo)
	for _, coinInfo := range coinInfos {
		CoinDeltas[createCoinKey(coinInfo.Symbol, coinInfo.Name)] = coinInfo
	}
	return CoinDeltas
}

func runCoinTask() {
	log.Debugf("runCoinTask: %s", time.Now().String())
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


		coinMapKey := createCoinKey(alert.CoinSymbol, alert.CoinName)
		coinInfo, ok := CoinDeltas[coinMapKey]
		if !ok {
			log.Error("Could not find coin with key", coinMapKey, " in api response map")
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
		// log.Debugf("CoinInfo %s: %s", alert.CoinSymbol, coinInfo)
		if (violation) {
			log.Debugf("Violation %s: (actual, threshold)=(%f, %f)",
				alert.CoinSymbol, change, alert.ThresholdDelta)
		}

		if (violation && noRecentViolations(alert.Email, coinInfo.Symbol, coinInfo.Name)) {

			lastUpdated, err := strconv.ParseInt(coinInfo.LastUpdated, 10, 64)

			if (err != nil) {
				log.Error(err.Error())
				lastUpdated = makeTimestamp()
			}

			notification := Notification{
				AlertId: alert.ID, Email: alert.Email, CoinName: coinInfo.Name, CoinSymbol: coinInfo.Symbol,
				TimeDelta: alert.TimeDelta, CurrentDelta: change, ThresholdDelta: alert.ThresholdDelta,
				LastUpdated: lastUpdated,
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

	log.Debug("done scanning active alerts from the alert table")
	log.Debugf("generating the following notifications:")
	printNotificationMap(notificationMap)

	// Send out the aggregated coin notification emails to user recipients.
	for email, notificationMap := range notificationMap {
		fmt.Printf("key[%s] value[%v]\n", email, notificationMap)
		res := sendNotificationsToUser(email, notificationMap)
		log.Debug(email, res)
	}
}

func checkTables() {
	log.Debug("alerts table:" + strconv.FormatBool(db.HasTable("alerts")))
	log.Debug("notifications table:" + strconv.FormatBool(db.HasTable("notifications")))
}


// Example format string. Everything except the message has a custom color
// which is dependent on the log level. Many fields have a custom output
// formatting too, eg. the time returns the hour down to the milli second.
var format = logging.MustStringFormatter(
	//`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.3s} %{color:reset} %{message}`,
)

func configureLogging() {
	errorBackend := logging.NewLogBackend(os.Stderr, "", 0)
	debugBackend := logging.NewLogBackend(os.Stdout, "", 0)

	// For messages written to backend2 we want to add some additional
	// information to the output, including the used log level and the name of
	// the function.
	debugBackendFormatter := logging.NewBackendFormatter(debugBackend, format)

	// Only errors and more severe messages should be sent to backend1
	errorBackendLeveled := logging.AddModuleLevel(errorBackend)
	errorBackendLeveled.SetLevel(logging.ERROR, "")

	// Set the backends to be used.
	logging.SetBackend(errorBackendLeveled, debugBackendFormatter)
	log.Debugf("configured logging successfully")
}

func main() {

	configureLogging()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// CORS default
	// Allows requests from any origin wth GET, HEAD, PUT, POST or DELETE method.
	// TODO: Don't use this in production.
	//e.Use(middleware.CORS())

	// CORS restricted
	// Allows requests from particular web origins.
	// with GET, PUT, POST or DELETE method.
	// ONLY ALLOW REQUESTS THAT ORIGINATE FROM THE WEBSITE (security risk).
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"https://cryptoalarms.com", "https://www.cryptoalarms.com"},
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
	}))

	// Sample hello world routes (for testing).
	e.GET("/api/hello", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "Hello, World!")
	})
	e.GET("/api/hello/:name", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "Hello, " + c.Param("name"))
	})

	// Routes for manipulating alerts.
	e.POST("/api/alerts", addAlert)
	e.POST("/api/alerts/delete", deleteAlert)
	e.GET("/api/alerts/:email", getAlerts)

	// Routes for manipulating notifications generated by alerts.
	e.GET("/api/notifications/:email", getNotifications)
	e.POST("/api/notifications/delete", deleteNotifications)
	e.GET("/api/notifications/count", countNotifications)
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

	// TODO: readd schedule
	scheduling := true
	if (scheduling) {
		var interval uint64
		interval = 30
		s := gocron.NewScheduler()
		s.Every(interval).Minutes().Do(runCoinTask)
		log.Debugf("scheduled alert task for every %d minutes", interval)
		s.Start()

	} else {
		runCoinTask() // runs the coin check task once.
	}

	// Start the web server.
	//port := ":9007"
	port := ":8443"
	fmt.Println("Started server on port $1", port)
	e.Logger.Error(e.Start(port))
}



