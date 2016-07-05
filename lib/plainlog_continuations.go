package insight_server

import (
	"fmt"
	"io"
	"path"
	"regexp"

	"bytes"
	"encoding/gob"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
)

type LogContinuation interface {

	// Returns the stored line to emit for a certain key.
	// Returns the line and a boolean indicating if there
	// was such a line in the DB
	HeaderLineFor(key []byte) ([]byte, bool, error)

	// Save the header for a certain key
	SetHeaderFor(key, value []byte) error

	// Removes any old entries from the DB
	VacuumOld(ttl time.Duration) error

	// we want to close this instance
	io.Closer
}

// Helper that returns a continuation key
func MakeContinuationKey(host, tsUtc, pid string) []byte {
	return []byte(fmt.Sprintf("%s||%s||%s", host, tsUtc, pid))
}

// ==================== Log continuation DB implementation ====================

var (
	logContinuationBucketName = []byte("log-continuations")
)

const (
	// The component name used during logging
	logContinuationComponentKey = "log-continuation-db"
	logContinuationDbFileName   = "log-continuations.db"

	// The TTL of the log continuations (as a time.Duration)
	// for now its 30 days
	logContinuationTTL = 30 * 24 * time.Hour

	// Scan for log continuation TTL this often
	logContinuationVacuumInterval = 24 * time.Hour
)

// A BoltDB-based implementation for the log continuation
type boltDbLogContinuation struct {
	db         *bolt.DB
	dbFileName string

	vacuumTicker *time.Ticker
}

type entryWithCreatedAt struct {
	CreatedAt time.Time
	Data      []byte
}

// Converts a bit of data to a GOB-bed data with a creation date
func encodeLogContinuation(data []byte) ([]byte, error) {
	now := time.Now().UTC()
	// GOB-encode the data
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	// Encode the data
	err := encoder.Encode(entryWithCreatedAt{now, data})
	return buffer.Bytes(), err
}

func decodeLogContinuation(r io.Reader) (*entryWithCreatedAt, error) {
	decoder := gob.NewDecoder(r)
	var e entryWithCreatedAt
	err := decoder.Decode(&e)
	return &e, err
}

func MakeBoltDbLogContinuationDb(dbDirectory string) (LogContinuation, error) {
	dbPath := path.Join(dbDirectory, logContinuationDbFileName)
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	// ensure that the continuation bucket exists
	err = db.Update(func(tx *bolt.Tx) error {
		// create the bucket if it does not exist
		_, err := tx.CreateBucketIfNotExists(logContinuationBucketName)
		if err != nil {
			return err
		}

		return nil
	})

	// Handler bucket creation errors
	if err != nil {
		return nil, fmt.Errorf("Error creating log continuation bucket: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"component": logContinuationComponentKey,
		"dir":       dbDirectory,
		"file":      dbPath,
	}).Info("Opened log continuation db")

	// Create the ticker for the vacuum job
	ticker := time.NewTicker(logContinuationVacuumInterval)

	// Create the output db here
	continuationDb := &boltDbLogContinuation{
		db:           db,
		dbFileName:   dbPath,
		vacuumTicker: ticker,
	}

	go func() {
		for range ticker.C {
			if err := continuationDb.VacuumOld(logContinuationTTL); err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{
					"component": logContinuationComponentKey,
				}).Error("Vaccum error")
			}
		}
	}()

	// create an actual instance
	return continuationDb, nil
}

// Returns the log continuation bucket from a transaction
func (l *boltDbLogContinuation) getBucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(logContinuationBucketName)
}

// Closes the database(file)
func (l *boltDbLogContinuation) Close() error {
	// stop the vacuum  ticker
	l.vacuumTicker.Stop()
	// Close the DB connection
	return l.db.Close()
}

func (l *boltDbLogContinuation) HeaderLineFor(key []byte) (value []byte, hasValue bool, err error) {
	value = nil
	hasValue = false

	err = l.db.View(func(tx *bolt.Tx) error {
		v := l.getBucket(tx).Get([]byte(key))

		// make a copy of the byte slice, as its deallocated
		// after the transaction
		if v != nil {
			c, err := decodeLogContinuation(bytes.NewReader(v))
			if err != nil {
				return fmt.Errorf("Error decoding log continuation GOB: %v", err)
			}

			// Ignore the TTL for now, as that is just here to keep the DB size down,
			// and does not provide any business logic

			// make a copy of the deserialized data (if the GOB deserializer
			// only references []byte-s during decode, we could end up with
			// an already deallocated slice here)
			hasValue = true
			buf := make([]byte, len(c.Data))
			copy(buf, c.Data)
			value = buf

		}
		return nil
	})

	return value, hasValue, err
}

// Save the header for a certain key
func (l *boltDbLogContinuation) SetHeaderFor(key, value []byte) error {

	data, err := encodeLogContinuation(value)
	if err != nil {
		return fmt.Errorf("Error serializing data to GOB: %v", err)
	}
	// Store the gob-encoded data for the key
	return l.db.Update(func(tx *bolt.Tx) error {
		return l.getBucket(tx).Put(key, data)
	})
}

// Clean up old entries
func (l *boltDbLogContinuation) VacuumOld(ttl time.Duration) error {
	// calculate the date that is the earliest valid date
	// for values
	earliestValidDate := time.Now().UTC().Add(-ttl)
	logrus.WithFields(logrus.Fields{
		"component":         logContinuationComponentKey,
		"earliestValidDate": earliestValidDate,
	}).Info("Starting vacuum on LogContinuationDb")

	// create a list of keys to delete (so we wont do deletition in
	// the foreach loop, but still keep it in this transaction)
	keysToDelete := make([][]byte, 0)

	// update the db
	return l.db.Update(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := l.getBucket(tx)

		// get the list of keys to remove
		err := b.ForEach(func(k, v []byte) error {
			// try to decode the entry
			c, err := decodeLogContinuation(bytes.NewReader(v))
			if err != nil {
				return fmt.Errorf("Error decoding log continuation GOB: %v", err)
			}

			// if decodable, check the TTL
			if c.CreatedAt.Before(earliestValidDate) {
				// if created before the earliest valid date, then delete this key
				keysToDelete = append(keysToDelete, k)
			}

			return nil
		})

		// check for errors
		if err != nil {
			return fmt.Errorf("Error while scanning db for log continuations: %v", err)
		}

		// Log the fact that we are cleaning up
		logrus.WithFields(logrus.Fields{
			"component": logContinuationComponentKey,
			"keys":      keysToDelete,
		}).Info("Deleting old keys")

		//delete the keys we have marked as too old
		for _, keyToDelete := range keysToDelete {
			if err := b.Delete(keyToDelete); err != nil {
				return fmt.Errorf("Error deleting old keys from log continations: %v", err)
			}
		}

		return nil
	})

}

// ==================== Line type checks ====================

var (
	lineContinuationRx         = regexp.MustCompile("logfile_rotation: opening new log")
	lineWillHaveContinuationRx = regexp.MustCompile("logfile_rotation: closing this log")
	lineHasPidRx               = regexp.MustCompile("^pid=([0-9]+)$")
)

// Returns true if the line string given looks like a continuation string
func IsLineContinuation(line string) bool {
	return lineContinuationRx.MatchString(line)
}

// Returns true if the line string given looks like a continuation string
func LineWillHaveContinuation(line string) bool {
	return lineWillHaveContinuationRx.MatchString(line)
}

// Returns true if the line is a PID-header line for plainlogs
func LineHasPid(line string) bool {
	return lineHasPidRx.MatchString(line)
}
