package kubeclients

import "errors"

var FieldOwnerNotSetError = errors.New("field owner option must be used when initializing KubeClients")
