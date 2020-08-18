package ipfs

import (
	"os/exec"
	"context"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/log"

	process "github.com/jbenet/goprocess"
)

type IpfsNode struct {
	proc  	  process.Process
	IsRunning bool
	Ready     chan bool
	cfg       config.Config
}

func NewIpsNode(cfg config.Config) *IpfsNode {
	return &IpfsNode{
		Ready: make(chan bool),
		cfg:   cfg,
	}
}

func (node *IpfsNode) Start(ctx context.Context) error {
    log.Info("Starting the ipfs node")

    proc := process.WithParent(process.Background())

    proc.Go(func(p process.Process) {
		cmd := exec.Command("space-ipfs-node")
		err := cmd.Run()
		if err != nil {
			log.Error("could not start the ipfs node", err)
			return
		}
    })

	log.Info("Running the ipfs node")

	node.proc = proc
	node.IsRunning = true
	node.Ready <- true

	return nil
}

func (node *IpfsNode) WaitForReady() chan bool {
	return node.Ready
}

func (node *IpfsNode) Stop() error {
	node.IsRunning = false
	node.proc.Close()

	close(node.Ready)

	return nil
}

func (node *IpfsNode) Shutdown() error {
	return node.Stop()
}
