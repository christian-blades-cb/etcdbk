package main // import "christian-blades-cb/etcdbk"

import (
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/jessevdk/go-flags"

	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"time"
)

func main() {
	var opts struct {
		EtcdMachines []string `long:"etcd-hosts" short:"e" required:"true" default:"http://127.0.0.1:2379" env:"ETCD_HOSTS" env-delim:"," description:"etcd machines"`
		ClusterName  string   `long:"cluster-name" short:"n" default:"etcd cluster" env:"CLUSTER_NAME"`

		AwsAccessKey string `long:"aws-access" env:"AWS_ACCESS_KEY"`
		AwsSecretKey string `long:"aws-secret" env:"AWS_SECRET_KEY"`
		AwsRegion    string `long:"aws-region" env:"AWS_REGION"`
		AwsBucket    string `long:"aws-bucket" env:"AWS_S3_BUCKET"`
	}
	flags.Parse(&opts)

	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)

	log.Debug("connecting to etcd")
	client := etcd.NewClient(opts.EtcdMachines)
	defer client.Close()
	log.Debug("connected")

	log.Debug("requesting root node")
	response, err := client.Get("/", false, true)
	if err != nil {
		log.WithField("error", err).Fatal("could not retrieve value for key")
	}

	tarBuf := bytes.NewBuffer(nil)
	w := tar.NewWriter(tarBuf)

	writeNode(w, response.Node)
	w.Close()

	tarBuf.WriteTo(os.Stdout)
}

func writeNode(w *tar.Writer, node *etcd.Node) { // I'm recursive!
	log.WithField("key", node.Key).Debug()
	if node.Dir {
		for _, subNode := range node.Nodes {
			writeNode(w, subNode) // see?
		}
		return
	}

	buf := bytes.NewBuffer([]byte(node.Value))
	expiration := func() string {
		if node.Expiration == nil {
			return "never"
		} else {
			return node.Expiration.Format(time.RFC3339)
		}
	}()

	w.WriteHeader(&tar.Header{
		Name: node.Key,
		Mode: 0444,
		Size: int64(buf.Len()),
		Xattrs: map[string]string{
			"ModifiedIndex": fmt.Sprintf("%d", node.ModifiedIndex),
			"CreatedIndex":  fmt.Sprintf("%d", node.CreatedIndex),
			"Expiration":    expiration,
		},
	})
	buf.WriteTo(w)
}
