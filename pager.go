package pager

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func Start(name string, arg ...string) (*Pager, error) {
	pager := &Pager{cmd: exec.Command(name, arg...)}
	if err := pager.start(); err != nil {
		return nil, err
	}
	return pager, nil
}

type Pager struct {
	cmd    *exec.Cmd
	stdout *os.File
}

func (p *Pager) start() error {
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	p.cmd.Stdin = r
	p.cmd.Stdout = os.Stdout
	if err := p.cmd.Start(); err != nil {
		return err
	}
	stdoutFd, err := syscall.Dup(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	p.stdout = os.NewFile(uintptr(stdoutFd), "stdout backup")
	if err := syscall.Dup2(int(w.Fd()), int(os.Stdout.Fd())); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

func (p *Pager) Wait() error {
	defer func() {
		os.Stdout = p.stdout
	}()
	os.Stdout.Close()
	exitCh := make(chan error)
	go func() {
		exitCh <- p.cmd.Wait()
	}()
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt)
	for {
		select {
		case sig := <-sigCh:
			p.cmd.Process.Signal(sig)
		case err := <-exitCh:
			os.Exit(1)
			if err != nil {
				return err
			} else {
				return nil
			}
		}
	}
}
