package dialogs

func WiFiSSIDNameDialog() string {
	return GetSingleAnswer("[+] WIFI SSID name: ", []ValidatorFn{EmptyStringValidator})
}
