package utils

import (
	"net"
)

func ClientIP() (ip string) {
	ift, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}

	for _, address := range ift {
		if addr, ok := address.(*net.IPNet); ok && !addr.IP.IsLoopback() {
			if addr.IP.To4() != nil {
				return addr.IP.String()
			}

		}
	}

	return
}

func NonDefaultLocalIP() (ip string) {
	ift, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	ip = "127.0.0.1"
	var addrs []net.Addr
AnotherLocalIPLoop:
	for _, iface := range ift {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		if addrs, err = iface.Addrs(); err != nil {
			panic(err)
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil && ipNet.IP.IsPrivate() {
					ip = ipNet.IP.String()
					break AnotherLocalIPLoop
				}
				if ipNet.IP.IsGlobalUnicast() {
					ip = ipNet.IP.String()
				}
			}
		}
	}
	return
}

func HostIPInDocker() (ip string) {
	addrs, err := net.LookupIP("host.docker.internal")
	if err != nil {
		return
	}

	for _, addr := range addrs {
		if addr.To4() != nil {
			return addr.String()
		}
	}
	return
}

type localIP struct {
	_     func()
	bytes []byte
	str   string
}

func (l *localIP) Bytes() (d []byte) {
	if l.bytes == nil {
		return
	}
	d = make([]byte, len(l.bytes), cap(l.bytes))
	copy(d, l.bytes)
	return
}

func (l *localIP) String() string {
	return l.str
}

var LocalIP = &localIP{
	str: ClientIP(),
	bytes: func() []byte {
		reverse := func(bs []byte) {
			i, j := 0, len(bs)-1
			for i < j {
				bs[i], bs[j] = bs[j], bs[i]
				i++
				j--
			}
		}

		if ipAddr := ClientIP(); ipAddr != "" {
			realIP := []byte("000000000000")
			idx := 0
			for i := len(ipAddr) - 1; i >= 0; i-- {
				c := ipAddr[i]
				if c == '.' {
					idx = (((idx - 1) / 3) + 1) * 3
					continue
				}
				realIP[idx] = c
				idx++
			}
			reverse(realIP)
			return realIP
		}
		return []byte("000000000000")
	}(),
}
