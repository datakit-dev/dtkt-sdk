package util

import (
	"fmt"
	"net"
	"strconv"
)

func GetFreePort(host string, portRange ...int) (int, error) {
	var min, max int
	if len(portRange) > 0 {
		min = portRange[0]
		if len(portRange) > 1 {
			max = portRange[1]
			if max < min {
				max = min
			}
		}
	}

	for port := min; port <= max; port++ {
		if IsPortFree(host, strconv.Itoa(port)) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no free port found in range %d-%d", min, max)
}

func IsPortFree(host, port string) bool {
	addr, err := net.ResolveTCPAddr("tcp", host+":"+port)
	if err == nil {
		lis, err := net.ListenTCP("tcp", addr)
		if err == nil {
			//nolint:errcheck
			lis.Close()
			return true
		}
	}
	return false
}
