package namedpipes

import (
	"context"
	"fmt"
	"sync"

	"github.com/Velocidex/ordereddict"
	"gopkg.in/natefinch/npipe.v2"
	"www.velocidex.com/golang/velociraptor/utils"
	"www.velocidex.com/golang/vfilter"
)

const (
	FREQUENCY = 30 // seconds
)

var (
	GlobalPipeService = NewPipeWatchService()
)

type PipeWatcherService struct {
	mu sync.Mutex

	registrations map[string][]*Handle
}

func NewPipeWatchService() *PipeWatcherService {
	return &PipeWatcherService{
		registrations: make(map[string][]*Handle),
	}
}

func (self *PipeWatcherService) Register(
	pipename string,
	ctx context.Context,
	scope vfilter.Scope,
	output_chan chan vfilter.Row) func() {

	self.mu.Lock()
	defer self.mu.Unlock()

	subctx, cancel := context.WithCancel(ctx)
	handle := &Handle{
		ctx:         subctx,
		output_chan: output_chan,
		scope:       scope,
	}

	registration, pres := self.registrations[pipename]
	if !pres {
		registration = []*Handle{}
		self.registrations[pipename] = registration

		go self.StartMonitoring(pipename, scope, ctx, output_chan)
	}

	registration = append(registration, handle)
	self.registrations[pipename] = registration

	scope.Log("Registered Named Pipe: %v", pipename)

	return cancel
}

func (self *PipeWatcherService) StartMonitoring(
	pipename string,
	scope vfilter.Scope,
	ctx context.Context,
	output_chan chan vfilter.Row) {

	defer utils.CheckForPanic("StartMonitoring")

	// Create the pipe.
	conn, err := npipe.Listen(fmt.Sprintf(`\\.\pipe\%s`, pipename))
	if err != nil {
		scope.Log("create_pipe: %v", err)
		return
	}
	scope.Log("Created Named Pipe: %s", fmt.Sprintf(`\\.\pipe\%s`, pipename))

	for {
		// Check for message.
		msg, err := conn.Accept()
		if err != nil {
			scope.Log("conn_accept: %v", err)
			continue
		}

		// Read the message into a buffer.
		buf := make([]byte, 256)
		size, err := msg.Read(buf)
		if err != nil || size <= 0 {
			scope.Log("msg_read: %v", err)
			continue
		}
		// Convert the message into a dictionary.
		event := ordereddict.NewDict().
			Set("Data", fmt.Sprintf("%x", buf))
		//scope.Log("msg: %v", event) // Testing purposes.

		// Output to our channel.
		select {
		case <-ctx.Done():
			return
		case output_chan <- event:
		}
	}
}

type Handle struct {
	ctx         context.Context
	output_chan chan vfilter.Row
	scope       vfilter.Scope
}
