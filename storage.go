package main

import (
	"bytes"
	"iter"

	bolt "go.etcd.io/bbolt"
)

type Repo interface {
	// Store url as processed
	Processed(url string) error
	// Store url as currently scheduled for processing
	Scheduled(url string) error
	// Return scheduled links
	ScheduledSeq() iter.Seq[string]
	// Return whether this url was already visited
	IsProcessed(url string) (bool, error)
	// Mark crawling process for startUrl started
	// Returns true if created task record and
	// returns false if crawl task already in progress
	StartCrawl(url string) (bool, error)
	// Mark crawling process as finished
	EndCrawl(url string) error
	// Returns whether Crawl task is already completed
	IsCrawlCompleted(url string) (bool, error)
	// Signals that repo should finish current in progress
	// work and close
	Close() error
}

var _ Repo = (*bboltRepo)(nil)

type bboltRepo struct {
	db *bolt.DB
}

var bucketProcessed []byte = []byte("processed")
var bucketScheduled []byte = []byte("scheduled")
var bucketTasks []byte = []byte("tasks")

func NewBboltRepo() (*bboltRepo, error) {
	db, err := bolt.Open("data/my.db", 0600, nil)

	if err != nil {
		return &bboltRepo{}, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketProcessed)

		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists(bucketScheduled)

		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists(bucketTasks)

		return err
	})

	if err != nil {
		return &bboltRepo{}, err
	}

	return &bboltRepo{db}, nil
}

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

func (b *bboltRepo) Scheduled(url string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketScheduled)

		return bucket.Put([]byte(url), []byte{})
	})
}

func (b *bboltRepo) ScheduledSeq() iter.Seq[string] {
	return func(yield func(string) bool) {
		b.db.View(func(tx *bolt.Tx) error {
			scheduled := tx.Bucket(bucketScheduled)

			cursor := scheduled.Cursor()

			for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
				if !yield(string(k)) {
					break
				}
			}

			return nil
		})
	}
}

type TaskStatus []byte

var (
	TaskInProgress TaskStatus = []byte("in_progress")
	TaskCompleted  TaskStatus = []byte("completed")
)

func (b *bboltRepo) StartCrawl(startUrl string) (bool, error) {
	var created bool = true
	err := b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketTasks)

		task := bucket.Get([]byte(startUrl))

		if task == nil {
			return bucket.Put([]byte(startUrl), TaskInProgress)
		}

		created = false
		return nil
	})

	return created, err
}

func (b *bboltRepo) EndCrawl(startUrl string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketTasks)

		return bucket.Put([]byte(startUrl), TaskCompleted)
	})
}

func (b *bboltRepo) IsCrawlCompleted(startUrl string) (bool, error) {
	var isCompleted bool

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketTasks)

		taskStatus := bucket.Get([]byte(startUrl))

		if bytes.Equal(taskStatus, TaskCompleted) {
			isCompleted = true
		}

		return nil
	})

	return isCompleted, err
}

func (b *bboltRepo) IsProcessed(url string) (bool, error) {
	var found bool
	err := b.db.View(func(tx *bolt.Tx) error {
		bProcessed := tx.Bucket(bucketProcessed)
		record := bProcessed.Get([]byte(url))
		found = record != nil

		return nil
	})

	return found, err
}

func (b *bboltRepo) Close() error {
	return b.db.Close()
}
