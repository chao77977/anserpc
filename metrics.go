package anserpc

import (
	"github.com/rcrowley/go-metrics"
)

var (
	_requestCounter = metrics.GetOrRegisterCounter(
		"anser/requests", nil)
	_successRequestCounter = metrics.GetOrRegisterCounter(
		"anser/success", nil)
	_failureReqeustCounter = metrics.GetOrRegisterCounter(
		"anser/failure", nil)
)
