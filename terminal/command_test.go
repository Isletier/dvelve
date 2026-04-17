package terminal

import (
	"os"
	"testing"

	"github.com/go-delve/delve/service/api"
)

func TestCommandDefault(t *testing.T) {
	var (
		cmds = Commands{}
		cmd  = cmds.Find("non-existent-command", noPrefix).cmdFn
	)

	err := cmd(nil, callContext{}, "")
	if err == nil {
		t.Fatal("cmd() did not default")
	}

	if err.Error() != "command not available" {
		t.Fatal("wrong command output")
	}
}

func TestCommandReplayWithoutPreviousCommand(t *testing.T) {
	var (
		cmds = DebugCommands(nil)
		cmd  = cmds.Find("", noPrefix).cmdFn
		err  = cmd(nil, callContext{}, "")
	)

	if err != nil {
		t.Error("Null command not returned", err)
	}
}

func TestCommandThread(t *testing.T) {
	var (
		cmds = DebugCommands(nil)
		cmd  = cmds.Find("thread", noPrefix).cmdFn
	)

	err := cmd(nil, callContext{}, "")
	if err == nil {
		t.Fatal("thread terminal command did not default")
	}

	if err.Error() != "you must specify a thread" {
		t.Fatal("wrong command output: ", err.Error())
	}
}

func TestIssue354(t *testing.T) {
	printStack(&Term{}, os.Stdout, []api.Stackframe{}, "", false)
	printStack(&Term{}, os.Stdout, []api.Stackframe{
		{Location: api.Location{PC: 0, File: "irrelevant.go", Line: 10, Function: nil},
			Bottom: true}}, "", false)
}
