
package native

import (
	"net"

	"github.com/mitre/gocat/execute"
)

func getIPAddresses(chOutput chan []byte, chStatus chan string) {
	output := []byte{}
	status := "0"

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		output = []byte("Error fetching IP addresses")
		status = execute.ERROR_STATUS
	} else {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ip_address := []byte(ipnet.IP.String() + "\n")
					output = append(output, ip_address...)
				}
			}
		}
	}

	chOutput <- output
	chStatus <- status
}