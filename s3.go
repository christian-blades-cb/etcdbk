package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"sync"
	"time"
)

type ToS3 struct {
	ClusterName string `long:"cluster-name" short:"n" default:"etcd-cluster" env:"CLUSTER_NAME" description:"Cluster name to use in naming the file in the S3 Bucket"`

	AwsAccessKey  string `long:"aws-access" env:"AWS_ACCESS_KEY_ID" description:"Access key of an IAM user with write access to the given bucket"`
	AwsSecretKey  string `long:"aws-secret" env:"AWS_SECRET_ACCESS_KEY" description:"Secret key of an IAM user with write access to the given bucket"`
	AwsS3Endpoint string `long:"s3-endpoint" env:"AWS_S3_ENDPOINT" default:"https://s3.amazonaws.com" description:"AWS S3 endpoint. See http://goo.gl/OG2Nkv"`
	AwsBucket     string `long:"aws-bucket" env:"AWS_S3_BUCKET" description:"Bucket in which to place the archive."`
}

var toS3 ToS3

func (o *ToS3) Execute(args []string) error {
	client := etcd.NewClient(opts.EtcdMachines)
	doSnapshot(client)
	return nil
}

type S3OnInterval struct {
	MaxPeriod         func(string) `long:"max-period" env:"MAX_PERIOD" description:"Longest time to wait between snapshots if there are no updates" default:"168h"`
	MinPeriod         func(string) `long:"min-period" env:"MIN_PERIOD" default:"1h" description:"How long to wait after an update to push the snapshot to S3"`
	MaxPeriodDuration time.Duration
	MinPeriodDuration time.Duration
}

var s3OnInterval S3OnInterval

func (o *S3OnInterval) Execute(args []string) error {
	client := etcd.NewClient(opts.EtcdMachines)
	events := make(chan *etcd.Response)
	go client.Watch("/", 0, true, events, nil)
	log.Info("listening for changes")

	var snapshotMutex sync.Mutex
	snapshotCondition := sync.NewCond(&snapshotMutex)
	go func() {
		for {
			snapshotCondition.L.Lock()
			snapshotCondition.Wait()

			log.Debug("snapshot triggered, waiting for minperiod")
			time.Sleep(o.MinPeriodDuration)

			log.Debug("minperiod expired, taking a snapshot")
			doSnapshot(client)
			snapshotCondition.L.Unlock()
		}
	}()

	maxPeriodTicker := time.NewTicker(o.MaxPeriodDuration)

	for {
		select {
		case <-maxPeriodTicker.C:
			log.Debug("maxperiod expired, taking snapshot")
			doSnapshot(client)
		case <-events:
			snapshotCondition.Signal()
		}
	}
}

func init() {
	s3OnInterval.MaxPeriod = func(dur string) {
		if pDur, err := time.ParseDuration(dur); err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"duration": dur,
			}).Fatal("could not parse maxperiod")
		} else {
			s3OnInterval.MaxPeriodDuration = pDur
		}
	}

	s3OnInterval.MinPeriod = func(dur string) {
		if pDur, err := time.ParseDuration(dur); err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"duration": dur,
			}).Fatal("could not parse minperiod")
		} else {
			s3OnInterval.MinPeriodDuration = pDur
		}
	}

	s3Cmd, _ := parser.AddCommand("s3",
		"Output to S3 bucket",
		"Output a tarball representing an etcd database into an S3 bucket",
		&toS3,
	)
	s3Cmd.AddCommand("continuous",
		"Backup to S3 continuously",
		"Backup an etcd database at regular intervals, or after changes",
		&s3OnInterval,
	)
}

func doSnapshot(client *etcd.Client) {
	log.Info("taking a snapshot")

	response, err := client.Get("/", false, true)
	if err != nil {
		log.WithField("error", err).Fatal("could not retrieve etcd root node")
	}
	buffer := FillTarballBuffer(response.Node)

	s3Writer := S3Writer{
		AccessKey:   toS3.AwsAccessKey,
		SecretKey:   toS3.AwsSecretKey,
		Endpoint:    toS3.AwsS3Endpoint,
		Bucket:      toS3.AwsBucket,
		ClusterName: toS3.ClusterName,
	}
	s3Writer.WriteToS3(buffer.Bytes())
	log.Info("wrote to bucket")
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
