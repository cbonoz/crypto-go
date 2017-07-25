package main

import (
	"fmt"
	"strings"
	"time"
)

func getHeadingRow(field string, value string) string {
	return fmt.Sprintf("<h4>%s: %s</h4>", field, value)
}
func getStringRow(field string, value string) string {
	return fmt.Sprintf("<b>%s</b>: %s<br/>", field, value)
}
func getFloatRow(field string, value float64) string {
	return fmt.Sprintf("<b>%s</b>: %.2f<br/>", field, value)
}
//func getIntRow(field string, value int64) string {
//	return fmt.Sprintf("<b>%s</b>: %s<br/>", field, value)
//}

func msToTime(msInt int64) (time.Time, error) {
	return time.Unix(0, msInt*1000*int64(time.Millisecond)), nil
}

func prettyPrintNotifications(alertNames []string, ns []Notification) string {
	var s []string
	for i := range ns {
		n := ns[i]
		alertName := alertNames[i]
		dateUpdated, err := msToTime(n.LastUpdated)
		if (err != nil) {
			log.Error(err)
		}

		nString := fmt.Sprintf("%s%s%s%s%s%s",
			getHeadingRow("Alert Name", alertName),
			getStringRow("Coin Name", n.CoinName),
			getStringRow("Coin Symbol", n.CoinSymbol),
			getFloatRow("Current % Change", n.CurrentDelta),
			getFloatRow("Threshold % Change", n.ThresholdDelta),
			getStringRow("As of time", dateUpdated.UTC().String()))
		s=append(s, nString)
	}
	return strings.Join(s, "<br/><hr/><br/>")
}

func createEmailBodyFromNotifications(alertNames []string, ns []Notification) string {

	const appName = "CryptoAlarms"
	const domain = "https://www.cryptoalarms.com/"
	const dashboardPage = domain + "dashboard"

	var numAlertString string
	numAlerts := len(ns)
	if (numAlerts > 1) {
		numAlertString = fmt.Sprintf("%d new alerts", numAlerts)
	} else {
		numAlertString = "a new alert"
	}
	alertString := prettyPrintNotifications(alertNames, ns)
	body := fmt.Sprintf(`
<body itemscope itemtype="http://schema.org/EmailMessage" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; -webkit-font-smoothing: antialiased; -webkit-text-size-adjust: none; width: 100%% !important; height: 100%%; line-height: 1.6em; background-color: #f6f6f6; margin: 0;" bgcolor="#f6f6f6">

<table class="body-wrap" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; width: 100%%; background-color: #f6f6f6; margin: 0;" bgcolor="#f6f6f6"><tr style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;"><td style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0;" valign="top"></td>
		<td class="container" width="600" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; display: block !important; max-width: 600px !important; clear: both !important; margin: 0 auto;" valign="top">
			<div class="content" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; max-width: 600px; display: block; margin: 0 auto; padding: 20px;">
				<table class="main" width="100%%" cellpadding="0" cellspacing="0" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; border-radius: 3px; background-color: #fff; margin: 0; border: 1px solid #e9e9e9;" bgcolor="#fff"><tr style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;"><td class="alert alert-warning" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 16px; vertical-align: top; color: #fff; font-weight: 500; text-align: center; border-radius: 3px 3px 0 0; background-color: #FF9F00; margin: 0; padding: 20px;" align="center" bgcolor="#FF9F00" valign="top">
				        Warning: New Currency Price Change Alert
						</td>
					</tr><tr style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;"><td class="content-wrap" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 20px;" valign="top">
							<table width="100%%" cellpadding="0" cellspacing="0" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;"><tr style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;"><td class="content-block" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
										You have <strong style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">%s</strong>.
									</td>
								</tr><tr id='main-content' style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;"><td class="content-block" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
								    %s
									</td>
								</tr>
								<tr style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;"><td class="content-block" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
										<a href="%s" class="btn-primary" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; color: #FFF; text-decoration: none; line-height: 2em; font-weight: bold; text-align: center; cursor: pointer; display: inline-block; border-radius: 5px; text-transform: capitalize; background-color: #348eda; margin: 0; border-color: #348eda; border-style: solid; border-width: 10px 20px;">View my account</a>
									</td>
								</tr><tr style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;"><td class="content-block" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
                                        Thanks for using <b>%s</b>.
									</td>
								</tr></table></td>
					</tr></table><div class="footer" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; width: 100%%; clear: both; color: #999; margin: 0; padding: 20px;">
					<table width="100%%" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;"><tr style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;"><td class="aligncenter content-block" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 12px; vertical-align: top; color: #999; text-align: center; margin: 0; padding: 0 0 20px;" align="center" valign="top">
					<a href="%s" style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 12px; color: #999; text-decoration: underline; margin: 0;">Modify</a> your Alert settings.</td>
						</tr></table></div></div>
		</td>
		<td style="font-family: 'Helvetica Neue',Helvetica,Arial,sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0;" valign="top"></td>
	</tr></table></body>`, numAlertString, alertString, dashboardPage, appName, dashboardPage)

	// return the email body html with the css code appended.
	return body + getCssCode()
}



