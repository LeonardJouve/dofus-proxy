package main

import "dofus-proxy/proxy"

func main() {
	p := proxy.New()
	p.Listen(10000, 1234)
}
