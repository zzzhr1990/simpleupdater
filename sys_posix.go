//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package simpleupdater

//this file attempts to contain all posix
//specific stuff, that needs to be implemented
//in some other way on other OSs... TODO!

import (
	"os"
	"os/exec"
	"syscall"
)

var (
	supported = true
	uid       = syscall.Getuid()
	gid       = syscall.Getgid()
	SIGUSR1   = syscall.SIGUSR1
	SIGUSR2   = syscall.SIGUSR2
	SIGTERM   = syscall.SIGTERM
)

func move(dst, src string) error {
	println("move ", dst, " ---> ", src)
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	if err != nil {
		println("move error: ", err.Error())
	}
	//HACK: we're shelling out to mv because linux
	//throws errors when crossing device boundaries.
	//TODO see sys_posix_mv.go
	cmd := exec.Command("mv", src, dst)
	if data, err := cmd.CombinedOutput(); err != nil {
		println("mv error: ", err.Error())
		println("mv comb output: ", string(data))
		return err
	}

	// Run sync to 'commit' the mv by clearing caches
	cmd2 := syncCmd()
	if red, err := cmd2.CombinedOutput(); err != nil {
		println("sync error: ", err.Error())
		println("sync comb output: ", string(red))
		return err
	}
	return nil
}

func syncCmd() *exec.Cmd {
	return exec.Command("sync")
}

func chmod(f *os.File, perms os.FileMode) error {
	return f.Chmod(perms)
}
func chown(f *os.File, uid, gid int) error {
	return f.Chown(uid, gid)
}
