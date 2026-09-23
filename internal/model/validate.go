package model

import (
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/util/validation"
)

// ValidateDNS1123Label checks value against the same rule Kubernetes
// enforces on object names (RFC 1123 label): lowercase alphanumeric
// characters or '-', max 63 chars, must start and end with an
// alphanumeric character. Used for identifiers that read as
// Kubernetes-style resource names to a human, even where this repo's
// own adapters hash them before actual use (kubestore, kuberbac) -
// kept strict so that stays true regardless of the adapter.
func ValidateDNS1123Label(field, value string) error {
	if errs := validation.IsDNS1123Label(value); len(errs) > 0 {
		return fmt.Errorf("%w: %s %q is invalid: %s", ErrInvalid, field, value, errs[0])
	}
	return nil
}

const maxSubjectLength = 253

// subjectPattern allows what a subject realistically is in this app: a
// plain identifier (the bootstrap admin's subject is literally "admin")
// or an email address - deliberately not full RFC 5322 validation, just
// a safe, bounded charset. Unlike ValidateDNS1123Label, '@' and mixed
// case are allowed, since Subject is never used as a raw Kubernetes
// object name (kubestore/kuberbac hash it) and this app's whole design
// assumes email-style subjects.
var subjectPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._%+\-@]*[A-Za-z0-9])?$`)

// ValidateSubject checks subject against subjectPattern and a length
// cap. Callers are expected to have already trimmed and non-empty
// checked subject.
func ValidateSubject(subject string) error {
	if len(subject) > maxSubjectLength {
		return fmt.Errorf("%w: subject must be at most %d characters", ErrInvalid, maxSubjectLength)
	}
	if !subjectPattern.MatchString(subject) {
		return fmt.Errorf("%w: subject %q contains characters that are not allowed", ErrInvalid, subject)
	}
	return nil
}
