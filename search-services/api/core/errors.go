package core

import "errors"

var ErrBadArguments = errors.New("arguments are not acceptable")
var ErrAlreadyExists = errors.New("resource or task already exists")
var ErrNotFound = errors.New("resource is not found")
var ErrUnauthorized = errors.New("user is unauthorized")
var ErrStarting = errors.New("error trying to start")
