package pager

import (
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

func Start(name string, arg ...string) (*Pager, error) {
	lock := &sync.Mutex{}
	pager := &Pager{
		cmd:  exec.Command(name, arg...),
		lock: lock,
		cond: sync.NewCond(lock),
	}
	if err := pager.start(); err != nil {
		return nil, err
	}
	return pager, nil
}

type Pager struct {
	lock       *sync.Mutex
	cmd        *exec.Cmd
	pipeR      *os.File
	realStdout int
	cond       *sync.Cond
	wait       *error
	pipeClose  *error
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
	// We cannot close the read end at this point. Otherwise we may get a SIGPIPE
	// if the pager terminates quickly after starting. So we keep a reference for
	// closing this later when we're sure the write end isn't referenced anymore.
	p.pipeR = r
	stdoutFd, err := syscall.Dup(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	p.realStdout = stdoutFd
	if err := syscall.Dup2(int(w.Fd()), int(os.Stdout.Fd())); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	exitCh := make(chan error)
	go func() {
		exitCh <- p.cmd.Wait()
	}()
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		defer signal.Stop(sigCh)
		for {
			select {
			case sig := <-sigCh:
				// Forward the signal to our pager, so he gets to decide what to do
				// with this.
				p.cmd.Process.Signal(sig)
				// Break the pipe to indicate that we are done sending data to the
				// pager. This allows users to jump to the bottom using F in less
				// and mirrors the way git log works.
				p.closePipe()
			case err := <-exitCh:
				closeErr := p.closePipe()
				p.cond.L.Lock()
				if closeErr != nil {
					p.wait = &closeErr
				} else {
					p.wait = &err
				}
				p.cond.L.Unlock()
				p.cond.Broadcast()
				return
			}
		}
	}()
	return nil
}

// closePipe
func (p *Pager) closePipe() error {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	if p.pipeClose != nil {
		return *p.pipeClose
	}
	devNull, err := os.Open(os.DevNull)
	p.pipeClose = &err
	if err != nil {
		return err
	}
	return syscall.Dup2(int(devNull.Fd()), int(os.Stdout.Fd()))
}

// Wait waits for the pager to exit. Can be called my multiple goroutines.
func (p *Pager) Wait() error {
	if err := p.closePipe(); err != nil {
		return err
	}
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	for p.wait == nil {
		p.cond.Wait()
	}
	if err := syscall.Dup2(p.realStdout, int(os.Stdout.Fd())); err != nil {
		return err
	}
	return *p.wait
}
