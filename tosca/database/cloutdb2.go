package database

import (
	"github.com/tliron/puccini/clout"

)

// CloutDB2 defines an implementation of CloutDB
type CloutDB2 struct {
    Dburl string
    
}

// NewCloutDb2 creates a CloutDB2 instance
func NewCloutDb2(dburl string) CloutDB {
	return CloutDB2 {dburl}
}

// Save method implementation of CloutDB interface for CloutDB2 instance
func (db CloutDB2) Save (clout *clout.Clout, urlString string, grammarVersion string) error {
    return nil
}