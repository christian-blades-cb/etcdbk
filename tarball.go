package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"time"
)

func FillTarballBuffer(rootNode *etcd.Node) *bytes.Buffer {
	buffer := bytes.NewBuffer(nil)

	gzipWriter := gzip.NewWriter(buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	// LIFO order; close the tar writer, then the gzip writer.
	defer gzipWriter.Close()
	defer tarWriter.Close()

	writeNode(tarWriter, rootNode)

	return buffer
}

func writeNode(w *tar.Writer, node *etcd.Node) { // I'm recursive!
	log.WithField("key", node.Key).Debug("writing to tarball")
	if node.Dir {
		// Always write a header for a directory, unless it's the root.
		if len(node.Key) > 0 {
			w.WriteHeader(&tar.Header{
				// Always strip the leading slash from the key.
				Name:   node.Key[1:] + "/",
				Mode:   0755,
				Xattrs: nodeXattrs(node),
			})
		}

		for _, subNode := range node.Nodes {
			writeNode(w, subNode) // see?
		}
		return
	}

	buf := bytes.NewBuffer([]byte(node.Value))
	w.WriteHeader(&tar.Header{
		// Always strip the leading slash from the key.
		Name:   node.Key[1:],
		Mode:   0644,
		Size:   int64(buf.Len()),
		Xattrs: nodeXattrs(node),
	})
	buf.WriteTo(w)
}

func nodeExpiration(node *etcd.Node) string {
	if node.Expiration == nil {
		return "never"
	} else {
		return node.Expiration.Format(time.RFC3339)
	}
}

func nodeXattrs(node *etcd.Node) map[string]string {
	return map[string]string{
		"ModifiedIndex": fmt.Sprintf("%d", node.ModifiedIndex),
		"CreatedIndex":  fmt.Sprintf("%d", node.CreatedIndex),
		"Expiration":    nodeExpiration(node),
	}
}
