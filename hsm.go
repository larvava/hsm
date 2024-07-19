package hsm

import (
	"context"
	"fmt"
)

const (
	EventError Event = "Error"
)

type StateMachine interface {
	Init(State, Event, any)
	Event(Event, any)
	State() (State, State)
	Close(cancelFunc context.CancelFunc)
}

type State string
type Event string

type Action func(any) error

// A map that serves the same function as a STT (State Transition Table).
type tKey struct {
	state State
	event Event
}
type trasitionMap map[tKey]*transition

type eventElement struct {
	event Event
	arg   any
}

// Represents a row in the STT (State Transition Table).
type transition struct {
	startState  State
	Event       Event
	targetState State
	Action      Action
}

// Creates a new transition. Used to build the transition map.
func NewTransition(start State, event Event, target State, action Action) *transition {
	return &transition{start, event, target, action}
}

// Implements a state machine based on the Boost FSM (Finite State Machine) model.
type hsm struct {
	trasitionMap trasitionMap
	prevState    State
	currentState State
	prevArg      any
	currentArg   any

	eventCh  chan eventElement
	isClosed bool
}

func NewStateMachine(ctx context.Context, errorHandler func(error), trasitions ...*transition) StateMachine {
	m := &hsm{
		trasitionMap: make(trasitionMap),
		eventCh:      make(chan eventElement, 0),
		isClosed:     false,
	}

	var finiteState = make(map[State]struct{})

	for _, trasition := range trasitions {
		finiteState[trasition.startState] = struct{}{}
		m.trasitionMap[tKey{trasition.startState, trasition.Event}] = trasition
	}

	//for state := range finiteState {
	//	m.trasitionMap[tKey{state, EventError}] =
	//		&transition{state, EventError, state, errorAction}
	//}

	go m.eventLoop(ctx, errorHandler)
	return m
}

// Handling Event
func (m *hsm) eventLoop(ctx context.Context, errorHandler func(error)) {
	for inputEvent := range m.eventCh {
		// Check if the event is defined for the current state
		key := tKey{m.currentState, inputEvent.event}
		trasition, exist := m.trasitionMap[key]
		if exist {
			fmt.Println(m.currentState, "-(", trasition.Event, ")->", trasition.targetState)
			m.prevState = m.currentState
			m.prevArg = m.currentArg
			m.currentState = trasition.targetState
			m.currentArg = inputEvent.arg

			err := trasition.Action(inputEvent.arg)
			if err != nil {
				//_ = m.trasitionMap[tKey{m.currentState, EventError}].Action(err.Error())
				errorHandler(err)
			}

		} else {
			select {
			case <-ctx.Done():
				m.isClosed = true
				close(m.eventCh)
				return
			default:
				//_ = m.trasitionMap[tKey{m.currentState, EventError}].Action(fmt.Sprintf("not found map key about state machine (state=%s, event=%s)", key.state, key.event))
				errorHandler(fmt.Errorf("not found map key about state machine (state=%s, event=%s)\n", key.state, key.event))
			}
		}
	}
}

// Set first state and evnet.
func (m *hsm) Init(initState State, initEvent Event, initArg any) {
	m.currentState = initState
	m.Event(initEvent, initArg)
}

func (m *hsm) Event(event Event, arg any) {
	if !m.isClosed {
		m.eventCh <- eventElement{event, arg}
	}
}

func (m *hsm) Close(cancelFunc context.CancelFunc) {
	m.isClosed = true
	close(m.eventCh)
	cancelFunc()
}

func (m *hsm) State() (prevState State, currentState State) {
	return m.prevState, m.currentState
}
