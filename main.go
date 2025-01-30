package main

import "dofus-proxy/proxy"

func main() {
	var proxy proxy.Proxy
	proxy.Listen(10000, 1234)
}
