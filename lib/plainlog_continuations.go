package insight_server

import (
	"fmt"
	"io"
	"path"
	"regexp"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
)

type LogContinuation interface {

	// Returns the stored line to emit for a certain key.
	// Returns the line and a boolean indicating if there
	// was such a line in the DB
	HeaderLineFor(key []byte) ([]byte, bool)

	// Save the header for a certain key
	SetHeaderFor(key, value []byte) error

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
	logContinuationDbFileName   = "log-continuation.db"
)

// A BoltDB-based implementation for the log continuation
type boltDbLogContinuation struct {
	db         *bolt.DB
	dbFileName string
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

	// create an actual instance
	return &boltDbLogContinuation{
		db:         db,
		dbFileName: dbPath,
	}, nil
}

// Returns the log continuation bucket from a transaction
func (l *boltDbLogContinuation) getBucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(logContinuationBucketName)
}

// Closes the database(file)
func (l *boltDbLogContinuation) Close() error {
	return l.db.Close()
}

func (l *boltDbLogContinuation) HeaderLineFor(key []byte) (value []byte, hasValue bool) {
	value = nil
	hasValue = false

	err := l.db.View(func(tx *bolt.Tx) error {
		v := l.getBucket(tx).Get([]byte(key))
		// make a copy of the byte slice, as its deallocated
		// after the transaction
		if v != nil {
			hasValue = true
			buf := make([]byte, len(v))
			copy(buf, v)
			value = buf
		}
		return nil
	})

	// there might be errors here if bolt has some
	// internal errors / IO errors, but generally we should
	// be safe for read-only transactions
	if err != nil {
		// FIXME: dont log it here, but pass this error upwards
		logrus.WithError(err).WithFields(logrus.Fields{
			"component": logContinuationComponentKey,
			"key":       key,
		}).Error("Error getting PID header from LogContinuationDb")

		// Signal that we dont have the value
		return nil, false
	}

	return value, hasValue
}

// Save the header for a certain key
func (l *boltDbLogContinuation) SetHeaderFor(key, value []byte) error {
	// Store the user model in the user bucket using the username as the key.
	return l.db.Update(func(tx *bolt.Tx) error {
		return l.getBucket(tx).Put(key, value)
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
