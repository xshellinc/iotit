package serialport

import "fmt"
import "log"

var defaultPort string

func GetPort(port string) (string, error) {
	if len(port) > 0 && port != "auto" {
		return port, nil
	}
	if defaultPort == "" {
		defaultPort = getDefaultPort()
		if defaultPort == "" {
			return "", fmt.Errorf("--port not specified and none were found")
		}
		log.Printf("Using port %s", defaultPort)
	}
	return defaultPort, nil
}
