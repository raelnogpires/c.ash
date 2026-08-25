package updater

import "os/exec"

type command interface{ Start() error }

func newCommand(name string, args ...string) command { return exec.Command(name, args...) }
