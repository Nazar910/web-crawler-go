package main

import (
	"bytes"
	"iter"

	bolt "go.etcd.io/bbolt"
)

type Repo interface {
	Processed(url string) error
	Scheduled(url string) error
	ScheduledSeq() iter.Seq[string]

	StartCrawl(url string) (bool, error)
	EndCrawl(url string) error
	IsCrawlCompleted(url string) (bool, error)
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

// Store url as processed
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

// Store url as currently scheduled for processing
func (b *bboltRepo) Scheduled(url string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketScheduled)

		return bucket.Put([]byte(url), []byte{})
	})
}

// Return scheduled links
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

// Mark crawling process for startUrl started
// Returns true if created task record and
// returns false if crawl task already in progress
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

// Mark crawling process as finished
func (b *bboltRepo) EndCrawl(startUrl string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketTasks)

		return bucket.Put([]byte(startUrl), TaskCompleted)
	})
}

// Returns whether Crawl task is already completed
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
