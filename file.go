package main

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"os"
	"strings"
)

type ToFile struct {
	OutFilePath string `long:"outfile" short:"o" env:"OUTFILE" description:"Where to write the resulting tarball (STDOUT if not set)"`
}

var toFile ToFile

func (o *ToFile) Execute(args []string) error {
	node := getRootNode(opts.EtcdMachines)
	buffer := FillTarballBuffer(&node)
	return writeToFile(buffer, o.OutFilePath)
}

func init() {
	parser.AddCommand("file",
		"Output to file",
		"Output a tarball representing the etcd database to a file on disk.",
		&toFile,
	)
}

func writeToFile(buff *bytes.Buffer, path string) error {
	trimmedPath := strings.TrimSpace(path)
	switch trimmedPath {
	case "-", "":
		if _, err := buff.WriteTo(os.Stdout); err != nil {
			log.WithField("error", err).Warn("could not write to stdout")
			return err
		}
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
