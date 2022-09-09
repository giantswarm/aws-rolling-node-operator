package key

type AnnotationsGetter interface {
	GetAnnotations() map[string]string
}

type LabelsGetter interface {
	GetLabels() map[string]string
}
