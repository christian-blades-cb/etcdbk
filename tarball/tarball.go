package tarball

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
	w := tar.NewWriter(gzipWriter)

	defer w.Close()
	defer gzipWriter.Close()

	writeNode(w, rootNode)

	return buffer
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
