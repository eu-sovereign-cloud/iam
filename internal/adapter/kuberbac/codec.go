package kuberbac

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// setSpec round-trips spec through JSON to set obj's "spec" field —
// simpler than sigs.k8s.io/structured-merge-diff-style field setters for
// these two small, flat spec shapes.
func setSpec(obj *unstructured.Unstructured, spec any) error {
	raw, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	obj.Object["spec"] = m
	return nil
}

func getSpec(obj *unstructured.Unstructured, out any) error {
	raw, err := json.Marshal(obj.Object["spec"])
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
