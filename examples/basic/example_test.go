package basic

import (
	"flag"
	"fmt"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
)

// BasicPlugin buffers all messages into a mailbox for this test.
type BasicPlugin struct {
	*network.Plugin
	Mailbox chan *messages.BasicMessage
}

func (state *BasicPlugin) Startup(net *network.Network) {
	// Create mailbox
	state.Mailbox = make(chan *messages.BasicMessage, 1)
}

func (state *BasicPlugin) Receive(ctx *network.MessageContext) error {
	switch msg := ctx.Message().(type) {
	case *messages.BasicMessage:
		state.Mailbox <- msg
	}
	return nil
}

// ExampleBasic demonstrates how to broadcast a message to a set of peers that discover
// each other through peer discovery.
func ExampleBasic() {
	flag.Parse()

	numNodes := 3

	host := "localhost"
	startPort := 5000

	var nodes []*network.Network
	var processors []*BasicPlugin

	for i := 0; i < numNodes; i++ {
		builder := builders.NewNetworkBuilder()
		builder.SetKeys(crypto.RandomKeyPair())
		builder.SetAddress(network.FormatAddress("kcp", host, uint16(startPort+i)))

		builder.AddPlugin(discovery.PluginID, new(discovery.Plugin))

		processors = append(processors, new(BasicPlugin))
		builder.AddPlugin("basic", processors[i])

		node, err := builder.Build()
		if err != nil {
			fmt.Println(err)
		}

		go node.Listen()

		// Bootstrap to Node 0.
		if i != 0 {
			node.Bootstrap(nodes[0].Address)
		}

		nodes = append(nodes, node)
	}

	// Wait for all nodes to finish discovering other peers.
	time.Sleep(1 * time.Second)

	// Broadcast out a message from Node 0.
	expected := "This is a broadcasted message from Node 0."
	nodes[0].Broadcast(&messages.BasicMessage{Message: expected})

	fmt.Println("Node 0 sent out a message.")

	// Check if message was received by other nodes.
	for i := 1; i < len(nodes); i++ {
		select {
		case received := <-processors[i].Mailbox:
			if received.Message != expected {
				fmt.Printf("Expected message %s to be received by node %d but got %v\n", expected, i, received.Message)
			} else {
				fmt.Printf("Node %d received a message from Node 0.\n", i)
			}
		case <-time.After(3 * time.Second):
			fmt.Printf("Timed out attempting to receive message from Node 0.\n")
		}
	}

	// Output:
	// Node 0 sent out a message.
	// Node 1 received a message from Node 0.
	// Node 2 received a message from Node 0.
}
