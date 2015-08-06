package main

import (
	"github.com/christian-blades-cb/etcdbk/tarball"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/jessevdk/go-flags"

	"fmt"
	"sync"
	"time"
)

var opts struct {
	EtcdMachines  []string `long:"etcd-hosts" short:"e" required:"true" default:"http://127.0.0.1:2379" env:"ETCD_HOSTS" env-delim:"," description:"etcd machines"`
	ClusterName   string   `long:"cluster-name" short:"n" default:"etcd-cluster" env:"CLUSTER_NAME" description:"Cluster name to use in naming the file in the S3 Bucket"`
	AwsAccessKey  string   `long:"aws-access" env:"AWS_ACCESS_KEY_ID" description:"Access key of an IAM user with write access to the given bucket"`
	AwsSecretKey  string   `long:"aws-secret" env:"AWS_SECRET_ACCESS_KEY" description:"Secret key of an IAM user with write access to the given bucket"`
	AwsS3Endpoint string   `long:"s3-endpoint" env:"AWS_S3_ENDPOINT" default:"https://s3.amazonaws.com" description:"AWS S3 endpoint. See http://goo.gl/OG2Nkv"`
	AwsBucket     string   `long:"aws-bucket" env:"AWS_S3_BUCKET" description:"Bucket in which to place the archive."`
	MaxPeriod     string   `long:"max-period" env:"MAX_PERIOD" description:"Longest time to wait between snapshots if there are no updates" default:"7d"`
	MinPeriod     string   `long:"min-period" env:"MIN_PERIOD" default:"1h" description:"How long to wait after an update to push the snapshot to S3"`

	Verbose bool `long:"debug" description:"verbose logging" env:"DEBUG"`
}

func main() {
	if _, err := flags.Parse(&opts); err != nil {
		log.WithField("error", err).Fatal("could not parse runtime options")
	}
	if opts.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	maxPeriodDuration, err := time.ParseDuration(opts.MaxPeriod)
	if err != nil {
		log.WithField("error", err).Fatal("unable to parse max period")
	}
	maxPeriodTicker := time.NewTicker(maxPeriodDuration)

	minPeriodDuration, err := time.ParseDuration(opts.MinPeriod)
	if err != nil {
		log.WithField("error", err).Fatal("unable to parse minimum period")
	}

	allEvents := make(chan *etcd.Response)
	etcdClient := etcd.NewClient(opts.EtcdMachines)
	etcdClient.Watch("/", 0, true, allEvents, nil)

	var doSnapshotCondition sync.Cond
	go func() {
		for {
			doSnapshotCondition.Wait()

			log.Debug("snapshot triggered, waiting for minperiod")
			time.Sleep(minPeriodDuration)

			log.Debug("minperiod expired, taking a snapshot")
			doSnapshot(etcdClient)
		}
	}()

	go func() {
		for {
			select {
			case <-maxPeriodTicker.C:
				doSnapshot(etcdClient)
			case <-allEvents:
				doSnapshotCondition.Signal()
			}
		}
	}()

}

func doSnapshot(client *etcd.Client) {
	log.Info("taking a snapshot")

	response, err := client.Get("/", false, true)
	if err != nil {
		log.WithField("error", err).Fatal("could not retrieve etcd root node")
	}
	buffer := tarball.FillTarballBuffer(response.Node)

	s3Writer := S3Writer{
		AccessKey:   opts.AwsAccessKey,
		SecretKey:   opts.AwsSecretKey,
		Endpoint:    opts.AwsS3Endpoint,
		Bucket:      opts.AwsBucket,
		ClusterName: opts.ClusterName,
	}
	s3Writer.WriteToS3(buffer.Bytes())
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
