package inventory

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/k8s/object"
)

// renderTag is the first line of the bytes the render digest hashes. A change
// to the encoding is a new tag, and a new tag changes every value a frontend
// has stored.
const renderTag = "opm-render-v1\n"

// errNotObject is the cause when an object's JSON is not a single JSON object.
var errNotObject = errors.New("the JSON is not an object")

// RenderDigest returns the render digest of objs, "sha256:" followed by 64
// lowercase hex digits. It reads only each object's JSON, never a CUE value,
// so a caller may drop its Resources after [object.Export] and digest later.
//
// For each object it decodes the JSON again, keeping every number's literal,
// sets the value of the labels.ManagedBy label to "" when metadata.labels
// holds that key, and encodes the result as JSON with sorted object keys, no
// HTML escaping and a trailing newline. Strings are written as Go's
// encoding/json Encoder writes them with HTML escaping off: U+2028, U+2029
// and control characters as \u escapes, invalid UTF-8 as U+FFFD. It hashes "opm-render-v1\n" followed
// by the encoded objects, sorted by group (from apiVersion), kind,
// metadata.namespace and metadata.name, with the encoded bytes as the final
// tie-break. A sort field that is missing or not a string reads as "".
//
// The managed-by value is the one value ignored: it is the runtime's name
// (opm-cli or opm-controller), so the cli and the operator digest one render
// equally, while any other change, including adding or removing the label,
// moves the digest (0012:D6). The empty set hashes the tag line alone.
//
// An object whose JSON is not a single JSON object fails with an error naming
// its index. RenderDigest never changes objs, their JSON or their Object.
func RenderDigest(objs []object.Exported) (string, error) {
	type encoded struct {
		group, kind, namespace, name string
		bytes                        []byte
	}
	all := make([]encoded, 0, len(objs))
	for i, o := range objs {
		obj, err := decodeObject(o.JSON)
		if err != nil {
			return "", fmt.Errorf("render digest: object %d: %w", i, err)
		}
		meta, _ := obj["metadata"].(map[string]any)
		if lbls, ok := meta["labels"].(map[string]any); ok {
			if _, ok := lbls[labels.ManagedBy]; ok {
				lbls[labels.ManagedBy] = ""
			}
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(obj); err != nil {
			return "", fmt.Errorf("render digest: object %d: %w", i, err)
		}
		apiVersion, _ := obj["apiVersion"].(string)
		group := ""
		if slash := strings.LastIndex(apiVersion, "/"); slash >= 0 {
			group = apiVersion[:slash]
		}
		kind, _ := obj["kind"].(string)
		namespace, _ := meta["namespace"].(string)
		name, _ := meta["name"].(string)
		all = append(all, encoded{group: group, kind: kind, namespace: namespace, name: name, bytes: buf.Bytes()})
	}

	slices.SortFunc(all, func(a, b encoded) int {
		return cmp.Or(
			cmp.Compare(a.group, b.group),
			cmp.Compare(a.kind, b.kind),
			cmp.Compare(a.namespace, b.namespace),
			cmp.Compare(a.name, b.name),
			bytes.Compare(a.bytes, b.bytes),
		)
	})

	h := sha256.New()
	h.Write([]byte(renderTag))
	for _, e := range all {
		h.Write(e.bytes)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// decodeObject decodes b as exactly one JSON object, keeping number literals.
func decodeObject(b []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var obj map[string]any
	if err := dec.Decode(&obj); err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errNotObject
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after the object")
	}
	return obj, nil
}
