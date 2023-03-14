package index

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
)

type StateStorage interface {
	LoadIndex() (*Index, error)
	SaveIndex(*Index) error
}

func NewStateStorage(stateFile string, ctx context.Context) (StateStorage, error) {
	if stateFile == "" {
		return &nullStorage{}, nil
	}
	uri, err := url.Parse(stateFile)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing cache URI")
	}
	switch uri.Scheme {
	case "":
		return newFileStorage(uri)
	case "gs":
		return newGcsStorage(uri, ctx)
	default:
		return nil, errors.New("Unknown cache URI type")
	}
}

// nullStorage
type nullStorage struct{}

func (c *nullStorage) LoadIndex() (*Index, error) {
	return NewIndex(), nil
}

func (c *nullStorage) SaveIndex(index *Index) error {
	return nil
}

// fileStorage
type fileStorage struct {
	path string
}

func newFileStorage(uri *url.URL) (*fileStorage, error) {
	return &fileStorage{
		path: uri.Path,
	}, nil
}

func (c *fileStorage) LoadIndex() (*Index, error) {
	index := NewIndex()
	inputFile, err := os.Open(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return index, nil
		}
		return nil, errors.WithStack(err)
	}

	if err := json.NewDecoder(inputFile).Decode(&index); err != nil {
		return nil, errors.WithStack(err)
	}
	return index, inputFile.Close()
}

func (c *fileStorage) SaveIndex(index *Index) error {
	log.Println("Saving cached index")
	f, err := os.Create(c.path)
	if err != nil {
		return errors.WithStack(err)
	}
	err = json.NewEncoder(f).Encode(index)
	if err != nil {
		return errors.WithStack(err)
	}
	return f.Close()
}

// gcsStorage
type gcsStorage struct {
	ctx    context.Context
	object *storage.ObjectHandle
}

func newGcsStorage(uri *url.URL, ctx context.Context) (*gcsStorage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	bucket := client.Bucket(uri.Host)
	if _, err := bucket.Attrs(ctx); err != nil {
		if err == storage.ErrBucketNotExist {
			return nil, errors.Wrapf(err, "Bucket %s does not exist", uri.Host)
		}
	}

	object := bucket.Object(strings.TrimLeft(uri.Path, "/"))

	return &gcsStorage{
		ctx:    ctx,
		object: object,
	}, nil
}

func (c *gcsStorage) LoadIndex() (*Index, error) {
	index := NewIndex()

	reader, err := c.object.NewReader(c.ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return index, nil
		}
		return nil, errors.WithStack(err)
	}

	if err := json.NewDecoder(reader).Decode(&index); err != nil {
		return nil, errors.WithStack(err)
	}
	return index, reader.Close()
}

func (c *gcsStorage) SaveIndex(index *Index) error {
	writer := c.object.NewWriter(c.ctx)
	if err := json.NewEncoder(writer).Encode(&index); err != nil {
		return errors.WithStack(err)
	}
	err := writer.Close()
	if err != nil {
		if err == storage.ErrBucketNotExist {
			return errors.Wrapf(err, "Bucket %s does not exist", c.object.BucketName())
		}
		return errors.WithStack(err)
	}

	return nil
}
