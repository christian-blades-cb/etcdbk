package main // import "github.com/christian-blades-cb/etcdbk"

import (
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/jessevdk/go-flags"
	"os"
)

var opts struct {
	EtcdMachines []string `long:"etcd-hosts" short:"e" required:"true" default:"http://127.0.0.1:4001" env:"ETCD_HOSTS" env-delim:"," description:"etcd machines"`
	Verbose      func()   `long:"debug" short:"v" description:"verbose logging"`
}

var parser = flags.NewParser(&opts, flags.Default)

func init() {
	log.SetOutput(os.Stderr)
	opts.Verbose = func() {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	if _, err := parser.Parse(); err != nil {
		log.Fatal("could not parse options")
	}
}

func getRootNode(machines []string) etcd.Node {
	log.WithField("etcdhosts", machines).Debug("connecting to etcd cluster")
	client := etcd.NewClient(machines)
	defer client.Close()

	log.Debug("requesting root node")
	response, err := client.Get("/", false, true)
	if err != nil {
		log.WithField("error", err).Fatal("could not retrieve value for key")
	}

	return *response.Node
}
