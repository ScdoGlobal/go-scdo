/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package node

import (
	"github.com/seeledevteam/slc/p2p"
	"github.com/seeledevteam/slc/rpc"
)

// Service represents a service which is registered to the node after the node starts.
type Service interface {
	// Protocols retrieves the P2P protocols the service wishes to start.
	Protocols() []p2p.Protocol

	APIs() (apis []rpc.API)

	Start(server *p2p.Server) error

	Stop() error
}
