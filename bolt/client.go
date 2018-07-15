package bolt

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
)

const (
	PhenomeBucket    = "PhenomeBucket"
	GenerationBucket = "GenerationBucket"
	PopulationBucket = "PopulationBucket"
)

var (
	buckets          = []string{PhenomeBucket, GenerationBucket, PopulationBucket}
	ErrUnknownBucket = errors.New("unknown bucket")
)

func New() (*Client, error) {
	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		return nil, err
	}

	for _, bucket := range buckets {
		err := db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return fmt.Errorf("create bucket: %s", err.Error())
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return &Client{db}, nil
}

type Client struct {
	db *bolt.DB
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) Update(bucket string, key []byte, v interface{}) error {
	if err := c.checkBucket(bucket); err != nil {
		return err
	}

	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(v); err != nil {
			return err
		}
		return b.Put(key, buf.Bytes())
	})
}

func (c *Client) Get(bucket string, key []byte, v interface{}) error {
	if err := c.checkBucket(bucket); err != nil {
		return err
	}

	return c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		buf := bytes.NewBuffer(b.Get(key))
		return gob.NewDecoder(buf).Decode(v)
	})
}

func (c *Client) checkBucket(name string) error {
	offset := -1
	for k, b := range buckets {
		if b == name {
			offset = k
			break
		}
	}
	if offset == -1 {
		return ErrUnknownBucket
	}
	return nil
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func btoi(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}
