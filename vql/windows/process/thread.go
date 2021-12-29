package process

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"github.com/Velocidex/ordereddict"
	"www.velocidex.com/golang/velociraptor/acls"
	vql_subsystem "www.velocidex.com/golang/velociraptor/vql"
	"www.velocidex.com/golang/velociraptor/vql/windows"
	"www.velocidex.com/golang/vfilter"
)

type ThreadLookupFunctionArgs struct {
	ThreadID uint32 `vfilter:"required,field=int,doc=Thread ID to lookup"`
}

type ThreadLookupFunction struct{}

func (self ThreadLookupFunction) Call(
	ctx context.Context,
	scope vfilter.Scope,
	args *ordereddict.Dict) vfilter.Any {

	err := vql_subsystem.CheckAccess(scope, acls.MACHINE_STATE)
	if err != nil {
		scope.Log("thread_lookup: %s", err)
		return 0
	}

	runtime.LockOSThread()

	defer vql_subsystem.CheckForPanic(scope, "threadlookup")

	arg := &ThreadLookupFunctionArgs{}
	err = vfilter.ExtractArgs(scope, args, arg)
	if err != nil {
		scope.Log("thread_lookup: %s", err.Error())
		return 0
	}

	pid, err := LookupThreadID(uint32(arg.ThreadID))

	return pid
}

func (self ThreadLookupFunction) Info(scope vfilter.Scope, type_map *vfilter.TypeMap) *vfilter.FunctionInfo {
	return &vfilter.FunctionInfo{
		Name:    "threadlookup",
		Doc:     "Lookup the Process ID that owns the Thread ID",
		ArgType: type_map.AddType(scope, &ThreadLookupFunctionArgs{}),
	}
}

func LookupThreadID(tid uint32) (uint32, error) {
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return 0, errors.New(
			fmt.Sprintf("CreateToolhelp32Snapshot failed to capture threads: %v", err))
	}
	defer windows.CloseHandle(handle)

	var thread windows.THREADENTRY32
	thread.Size = uint32(unsafe.Sizeof(thread))
	err = windows.Thread32First(handle, &thread)
	if err != nil {
		return 0, errors.New(
			fmt.Sprintf("Failed to find first thread: %v", err))
	}

	var pid uint32 = 0
	for {
		if tid == thread.ThreadID {
			pid = thread.ProcessID
			break
		}

		err = windows.Thread32Next(handle, &thread)
		if err != nil {
			return pid, errors.New(
				fmt.Sprintf("Failed to find next thread: %v", err))
		}
	}

	return pid, nil
}

func init() {
	vql_subsystem.RegisterFunction(&ThreadLookupFunction{})
}
