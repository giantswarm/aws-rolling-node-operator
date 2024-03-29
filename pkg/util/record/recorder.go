// Package record implements recording functionality.
package record

import (
	"sync"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

var (
	initOnce        sync.Once
	defaultRecorder record.EventRecorder
)

func init() {
	defaultRecorder = new(record.FakeRecorder)
}

// InitFromRecorder initializes the global default recorder. It can only be called once.
// Subsequent calls are considered noops.
func InitFromRecorder(recorder record.EventRecorder) {
	initOnce.Do(func() {
		defaultRecorder = recorder
	})
}

// Event constructs an event from the given information and puts it in the queue for sending.
func Event(object runtime.Object, reason, message string) {
	defaultRecorder.Event(object, corev1.EventTypeNormal, cases.Title(language.Und, cases.NoLower).String(reason), message)
}

// Eventf is just like Event, but with Sprintf for the message field.
func Eventf(object runtime.Object, reason, message string, args ...interface{}) {
	defaultRecorder.Eventf(object, corev1.EventTypeNormal, cases.Title(language.Und, cases.NoLower).String(reason), message, args...)
}

// Warn constructs a warning event from the given information and puts it in the queue for sending.
func Warn(object runtime.Object, reason, message string) {
	defaultRecorder.Event(object, corev1.EventTypeWarning, cases.Title(language.Und, cases.NoLower).String(reason), message)
}

// Warnf is just like Warn, but with Sprintf for the message field.
func Warnf(object runtime.Object, reason, message string, args ...interface{}) {
	defaultRecorder.Eventf(object, corev1.EventTypeWarning, cases.Title(language.Und, cases.NoLower).String(reason), message, args...)
}
