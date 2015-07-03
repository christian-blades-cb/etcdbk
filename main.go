package main // import "github.com/christian-blades-cb/etcdbk"

import (
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/jessevdk/go-flags"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"

	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func main() {
	var opts struct {
		EtcdMachines []string `long:"etcd-hosts" short:"e" required:"true" default:"http://127.0.0.1:2379" env:"ETCD_HOSTS" env-delim:"," description:"etcd machines"`
		ClusterName  string   `long:"cluster-name" short:"n" default:"etcd-cluster" env:"CLUSTER_NAME" description:"Cluster name to use in naming the file in the S3 Bucket"`

		OutFilePath string `long:"outfile" short:"o" env:"OUTFILE" description:"Where to write the resulting tarball. '-' for STDOUT"`

		AwsAccessKey  string `long:"aws-access" env:"AWS_ACCESS_KEY_ID" description:"Access key of an IAM user with write access to the given bucket"`
		AwsSecretKey  string `long:"aws-secret" env:"AWS_SECRET_ACCESS_KEY" description:"Secret key of an IAM user with write access to the given bucket"`
		AwsS3Endpoint string `long:"s3-endpoint" env:"AWS_S3_ENDPOINT" default:"https://s3.amazonaws.com" description:"AWS S3 endpoint. See http://goo.gl/OG2Nkv"`
		AwsBucket     string `long:"aws-bucket" env:"AWS_S3_BUCKET" description:"Bucket in which to place the archive."`
	}
	flags.Parse(&opts)

	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)

	log.Debug("connecting to etcd")
	client := etcd.NewClient(opts.EtcdMachines)
	defer client.Close()

	log.Debug("requesting root node")
	response, err := client.Get("/", false, true)
	if err != nil {
		log.WithField("error", err).Fatal("could not retrieve value for key")
	}

	// fill the buffer with tarball
	buffer := bytes.NewBuffer(nil)
	gzipWriter := gzip.NewWriter(buffer)
	w := tar.NewWriter(gzipWriter)

	writeNode(w, response.Node)

	w.Close()
	gzipWriter.Close()

	// write to file
	log.WithField("outpath", opts.OutFilePath).Debug("writing to destination")
	outWriter, err := getOutfileWriter(opts.OutFilePath)
	if err != nil {
		log.WithField("error", err).Fatal("error opening output file")
	}
	defer outWriter.Close()
	if _, err := buffer.WriteTo(outWriter); err != nil {
		log.WithField("error", err).Warn("Error writing to file")
	}

	log.WithField("bucket", opts.AwsBucket).Debug("saving to bucket")
	// write to S3
	s3Out := S3Writer{
		AccessKey:   opts.AwsAccessKey,
		SecretKey:   opts.AwsSecretKey,
		Endpoint:    opts.AwsS3Endpoint,
		Bucket:      opts.AwsBucket,
		ClusterName: opts.ClusterName,
	}
	s3Out.WriteToS3(buffer.Bytes())
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

type DiscardCloser struct{}

func (d *DiscardCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

func (d *DiscardCloser) Close() error {
	return nil
}

func getOutfileWriter(path string) (io.WriteCloser, error) {
	trimmedPath := strings.TrimSpace(path)
	switch trimmedPath {
	case "-":
		return os.Stdout, nil
	case "":
		return &DiscardCloser{}, nil
	}
	return os.Create(path)
}

type S3Writer struct {
	AccessKey, SecretKey, Endpoint, Bucket string

	ClusterName string
}

func (s3w S3Writer) WriteToS3(p []byte) error {
	auth := aws.Auth{
		AccessKey: s3w.AccessKey,
		SecretKey: s3w.SecretKey,
	}
	path := fmt.Sprintf("%s-%s.tar.gz", s3w.ClusterName, time.Now().UTC().Format(time.RFC3339))

	client := s3.New(auth, aws.Region{S3Endpoint: s3w.Endpoint})
	bucket := client.Bucket(s3w.Bucket)
	return bucket.Put(path, p, "application/x-gzip", s3.Private, s3.Options{})
}
