package main

import (
	bolt "go.etcd.io/bbolt"
)

type Repo interface {
	Processed(url string) error
	Scheduled(url string) error
}

var _ Repo = (*bboltRepo)(nil)

type bboltRepo struct {
	db *bolt.DB
}

var bucketProcessed []byte = []byte("processed")
var bucketScheduled []byte = []byte("scheduled")

func NewBboltRepo() (*bboltRepo, error) {
	db, err := bolt.Open("data/my.db", 0600, nil)

	if err != nil {
		return &bboltRepo{}, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists(bucketProcessed)
		tx.CreateBucketIfNotExists(bucketScheduled)

		return nil
	})

	if err != nil {
		return &bboltRepo{}, err
	}

	return &bboltRepo{db}, nil
}

// store url as processed
func (b *bboltRepo) Processed(url string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bsched := tx.Bucket(bucketScheduled)
		err := bsched.Delete([]byte(url))

		if err != nil {
			return err
		}

		bproc := tx.Bucket(bucketProcessed)
		return bproc.Put([]byte(url), []byte{})
	})
}

// store url as currently scheduled for processing
func (b *bboltRepo) Scheduled(url string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketScheduled)

		return bucket.Put([]byte(url), []byte{})
	})
}
