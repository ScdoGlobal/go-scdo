/**
*  @file
*  @copyright defined in slc/LICENSE
 */

package node

import (
	"crypto/ecdsa"

	"github.com/seelecredoteam/go-seelecredo/common"
	"github.com/seelecredoteam/go-seelecredo/core"
	"github.com/seelecredoteam/go-seelecredo/log/comm"
	"github.com/seelecredoteam/go-seelecredo/metrics"
	"github.com/seelecredoteam/go-seelecredo/p2p"
)

// Config is the Configuration of node
type Config struct {
	//Config is the Configuration of log
	LogConfig comm.LogConfig

	// basic config for Node
	BasicConfig BasicConfig

	// The configuration of p2p network
	P2PConfig p2p.Config

	// HttpServer config for http server
	HTTPServer HTTPServer

	// The SeeleCredoConfig is the configuration to create the seeleCredo service.
	SeeleCredoConfig SeeleCredoConfig

	// The configuration of websocket rpc service
	WSServerConfig WSServerConfig

	// The configuration of ipc rpc service
	IpcConfig IpcConfig

	// metrics config info
	MetricsConfig *metrics.Config
}

// IpcConfig config for ipc rpc service
type IpcConfig struct {
	PipeName string `json:"name"`
}

// BasicConfig config for Node
type BasicConfig struct {
	// The name of the node
	Name string `json:"name"`

	// The version of the node
	Version string `json:"version"`

	// The file system path of the node, used to store data
	DataDir string `json:"dataDir"`

	// The file system path of the temporary dataset, used for spow
	DataSetDir string `json:"dataSetDir"`

	// RPCAddr is the address on which to start RPC server.
	RPCAddr string `json:"address"`

	// coinbase used by the miner
	Coinbase string `json:"coinbase"`

	// privatekey for coinbase, used in bft consensus
	PrivateKey string `json:"privateKey"`

	// MinerAlgorithm miner algorithm
	MinerAlgorithm string `json:"algorithm"`
}

// HTTPServer config for http server
type HTTPServer struct {
	// The HTTPAddr is the address of HTTP rpc service
	HTTPAddr string `json:"address"`

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors []string `json:"crossorigins"`

	// HTTPHostFilter is the whitelist of hostnames which are allowed on incoming requests.
	HTTPWhiteHost []string `json:"whiteHost"`
}

// WSServerConfig config for websocket server
type WSServerConfig struct {
	// The Address is the address of Websocket rpc service
	Address string `json:"address"`

	CrossOrigins []string `json:"crossorigins"`
}

// Config is the seeleCredo's configuration to create seeleCredo service
type SeeleCredoConfig struct {
	TxConf core.TransactionPoolConfig

	Coinbase common.Address

	CoinbasePrivateKey *ecdsa.PrivateKey

	GenesisConfig core.GenesisInfo
}

func (conf *Config) Clone() *Config {
	cloned := *conf
	if conf.MetricsConfig != nil {
		temp := *conf.MetricsConfig
		cloned.MetricsConfig = &temp
	}

	return &cloned
}
