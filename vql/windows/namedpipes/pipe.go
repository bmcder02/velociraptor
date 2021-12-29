package namedpipes

import (
	"context"

	"github.com/Velocidex/ordereddict"
	vql_subsystem "www.velocidex.com/golang/velociraptor/vql"
	vfilter "www.velocidex.com/golang/vfilter"
	"www.velocidex.com/golang/vfilter/arg_parser"
)

type _WatchPipePluginArgs struct {
	PipeName string `vfilter:"required,field=pipe_name,doc=The name of the named pipe."`
}

type _WatchPipePlugin struct{}

func (self _WatchPipePlugin) Call(
	ctx context.Context,
	scope vfilter.Scope,
	args *ordereddict.Dict) <-chan vfilter.Row {

	output_chan := make(chan vfilter.Row)

	go func() {
		defer close(output_chan)

		arg := &_WatchPipePluginArgs{}
		err := arg_parser.ExtractArgsWithContext(ctx, scope, args, arg)
		if err != nil {
			scope.Log("parse_lines: %v", err)
			return
		}

		event_channel := make(chan vfilter.Row)

		cancel := GlobalPipeService.Register(
			arg.PipeName,
			ctx, scope, event_channel)
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-event_channel:
				if !ok {
					scope.Log("Not okay. Exitting.")
					return
				}
				output_chan <- event
			}
		}
	}()

	return output_chan
}

func (self _WatchPipePlugin) Info(
	scope vfilter.Scope,
	type_map *vfilter.TypeMap) *vfilter.PluginInfo {
	return &vfilter.PluginInfo{
		Name:    "watch_pipe",
		Doc:     "Create and watch a named pipe",
		ArgType: type_map.AddType(scope, &_WatchPipePluginArgs{}),
	}
}

func init() {
	vql_subsystem.RegisterPlugin(&_WatchPipePlugin{})
}
