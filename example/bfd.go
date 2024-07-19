package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/larvava/hsm"
)

const (
	Down      hsm.State = "Down"
	Up        hsm.State = "Up"
	AdminDown hsm.State = "AdminDown"
	Init      hsm.State = "Init"

	EventDown      hsm.Event = "Down"
	EventUp        hsm.Event = "Up"
	EventAdminDown hsm.Event = "AdminDown"
	EventInit      hsm.Event = "Init"
	EventTimeOut   hsm.Event = "TimedOut"
	EventExit      hsm.Event = "Exit"
)

func initAction(arg any) error {
	fmt.Println("An [Init] packet is sent to initiate a BFD session.")
	return nil
}

func upAction(arg any) error {
	fmt.Println("An [Up] packet is sent. The session is normal.")
	return nil
}

func downAction(arg any) error {
	*arg.(*uint)++
	fmt.Printf("A [Down] packet is sent. The session is abnormal. (downCnt=%d)\n", *arg.(*uint))
	return nil
}

func adminDownAction(arg any) error {
	*arg.(*uint)++
	fmt.Printf("An [Admin down] packet is sent to indicate that it has been down by the administrator. (downCnt=%d)\n", *arg.(*uint))
	return nil
}

func errorHandler(err error) {
	fmt.Println("Error : ", err.Error())
}

func main() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer wg.Done()
		for {
			if ctx.Err() != nil {
				log.Fatalln("Closing state machine.")
			}
		}
	}()

	wg.Add(1)
	hsm := hsm.NewStateMachine(ctx, errorHandler,
		hsm.NewTransition(Init, EventDown, Init, initAction),
		hsm.NewTransition(Init, EventTimeOut, Down, downAction),
		hsm.NewTransition(Init, EventInit, Up, upAction),
		hsm.NewTransition(Init, EventUp, Up, upAction),
		// -----------------------------------------------------------------
		hsm.NewTransition(Up, EventInit, Up, upAction),
		hsm.NewTransition(Up, EventUp, Up, upAction),
		hsm.NewTransition(Up, EventDown, Down, downAction),
		hsm.NewTransition(Up, EventTimeOut, Down, downAction),
		hsm.NewTransition(Up, EventAdminDown, AdminDown, adminDownAction),
		// -----------------------------------------------------------------
		hsm.NewTransition(Down, EventInit, Up, upAction),
		hsm.NewTransition(Down, EventUp, Down, upAction),
		hsm.NewTransition(Down, EventDown, Init, upAction),
		// -----------------------------------------------------------------
		hsm.NewTransition(AdminDown, EventExit, Up, upAction),
	)

	var downCount = new(uint)
	*downCount = 0

	// Test a virtual bfd session
	hsm.Init(Init, EventInit, downCount)

	hsm.Event(EventUp, nil)
	hsm.Event(EventDown, downCount)
	hsm.Event(EventInit, nil)

	hsm.Event(EventUp, nil)
	hsm.Event(EventUp, nil)
	hsm.Event(EventUp, nil)
	hsm.Event(EventUp, nil)
	hsm.Event(EventUp, nil)

	hsm.Event(EventDown, downCount)

	// It is an event that cannot be received. An error will occur.
	hsm.Event(EventAdminDown, downCount)

	// Receive new event after executing error action.
	hsm.Event(EventUp, nil)
	hsm.Event(EventUp, nil)
	hsm.Event(EventUp, nil)
	hsm.Event(EventUp, nil)

	// This should end here
	hsm.Close(cancel)
	wg.Done()

	// These events will not be accepted.
	hsm.Event(EventInit, downCount)
	hsm.Event(EventUp, nil)
	hsm.Event(EventDown, downCount)
	hsm.Event(EventAdminDown, downCount)

	wg.Wait()
}
