package examples_test

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/foomo/shop/examples"
	"github.com/foomo/shop/mock"
)

func ExampleSmurfProcessor() {
	log.Println("runtime.GOMAXPROCS(16)", runtime.GOMAXPROCS(16))
	q := mock.GetMockQueue()

	const (
		pete = "pete"
		joe  = "joe"
	)

	// add some products in status a

	smurfOrders := map[string]int{
		pete: 100,
		joe:  200,
	}

	p := mock.GetMockPersistor("queue_test")
	numberOfOrders := 0
	for smurf, smurfOrderCount := range smurfOrders {
		for i := 0; i < smurfOrderCount; i++ {
			o := mock.MakeMockOrder(smurf)
			p.InsertOrder(o)
			numberOfOrders++
			if numberOfOrders%100 == 0 {
				log.Println(smurf, numberOfOrders)
			}
		}
	}

	log.Println("done writing orders")

	start := time.Now()

	joeProcessor := examples.NewSmurfProcessor(joe)
	peteProcessor := examples.NewSmurfProcessor(pete)

	chanDone := make(chan string)

	a := func() {
		q.RunProcessor(peteProcessor)
		chanDone <- "pete"
	}

	b := func() {
		q.RunProcessor(joeProcessor)
		chanDone <- "joe"
	}
	go a()
	go b()
	log.Println(<-chanDone)
	log.Println(<-chanDone)
	log.Println(time.Now().Sub(start))
	log.Println("done processing")

	fmt.Println("number of orders:", numberOfOrders, ", processed by joe:", joeProcessor.CountProcessed, ", processed by pete:", peteProcessor.CountProcessed)
	// Output: number of orders: 300 , processed by joe: 200 , processed by pete: 100

}
