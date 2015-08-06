package main // import "github.com/christian-blades-cb/etcdbk"

import (
	"github.com/christian-blades-cb/etcdbk/tarball"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/jessevdk/go-flags"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"

	"bytes"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	var opts struct {
		EtcdMachines []string `long:"etcd-hosts" short:"e" required:"true" default:"http://127.0.0.1:2379" env:"ETCD_HOSTS" env-delim:"," description:"etcd machines"`

		OutFilePath string `long:"outfile" short:"o" env:"OUTFILE" description:"Where to write the resulting tarball. '-' for STDOUT"`

		ClusterName   string `long:"cluster-name" short:"n" default:"etcd-cluster" env:"CLUSTER_NAME" description:"Cluster name to use in naming the file in the S3 Bucket"`
		AwsAccessKey  string `long:"aws-access" env:"AWS_ACCESS_KEY_ID" description:"Access key of an IAM user with write access to the given bucket"`
		AwsSecretKey  string `long:"aws-secret" env:"AWS_SECRET_ACCESS_KEY" description:"Secret key of an IAM user with write access to the given bucket"`
		AwsS3Endpoint string `long:"s3-endpoint" env:"AWS_S3_ENDPOINT" default:"https://s3.amazonaws.com" description:"AWS S3 endpoint. See http://goo.gl/OG2Nkv"`
		AwsBucket     string `long:"aws-bucket" env:"AWS_S3_BUCKET" description:"Bucket in which to place the archive."`

		Verbose bool `long:"debug" description:"verbose logging"`
	}
	if _, err := flags.Parse(&opts); err != nil {
		log.Fatal("could not parse options")
	}

	log.SetOutput(os.Stderr)
	if opts.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	if opts.OutFilePath == "" && opts.AwsBucket == "" {
		log.Fatal("No file path or S3 bucket given. Nothing to do.")
	}

	rootNode := getRootNode(opts.EtcdMachines)
	buffer := tarball.FillTarballBuffer(&rootNode)

	// write to file
	log.WithField("outpath", opts.OutFilePath).Debug("writing to destination")
	writeToFile(buffer, opts.OutFilePath)

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

func writeToFile(buff *bytes.Buffer, path string) error {
	trimmedPath := strings.TrimSpace(path)
	switch trimmedPath {
	case "-":
		if _, err := buff.WriteTo(os.Stdout); err != nil {
			log.WithField("error", err).Warn("could not write to stdout")
			return err
		}
	case "":
		log.Debug("empty path. not writing.")
	default:
		wc, err := os.Create(path)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"filepath": path,
			}).Warn("could not create file for writing")
			return err
		}
		defer wc.Close()

		if _, err := buff.WriteTo(wc); err != nil {
			log.WithField("error", err).Warn("could not write to file")
			return err
		}
	}

	return nil
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
