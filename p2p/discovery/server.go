/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package discovery

import (
	"net"

	"github.com/seeledevteam/slc/common"
)

// StartService start node udp service
func StartService(nodeDir string, myID common.Address, myAddr *net.UDPAddr, bootstrap []*Node, shard uint) *Database {
	udp := newUDP(myID, myAddr, shard)

	if bootstrap != nil {
		udp.trustNodes = bootstrap
	}
	udp.loadNodes(nodeDir)
	udp.StartServe(nodeDir)

	return udp.db
}
