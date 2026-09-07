package application

import "context"

type Record struct {
	Scope          string
	Key            string
	Fingerprint    string
	Status         string
	ResponseStatus int
	ResponseBody   []byte
}

type Repository interface {
	Reserve(context.Context, Record) (bool, error)
	Find(context.Context, string, string) (Record, error)
	Complete(context.Context, Record) error
	Release(context.Context, string, string, string) error
}
