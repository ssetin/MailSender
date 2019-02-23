// test comment2
package main

import "fmt"

var mdstr MailDistributor

func main() {
	err := mdstr.Init()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	mdstr.Start()

	mdstr.Close()
}
