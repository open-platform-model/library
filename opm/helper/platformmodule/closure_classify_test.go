package platformmodule

import (
	"context"
	"errors"
	"testing"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

// errSource answers every module-file request with one error.
type errSource struct{ err error }

func (s errSource) ModFile(context.Context, module.Version) (*modfile.File, error) {
	return nil, s.err
}

// fetch-error-classification spec: the dependency closure classifies the
// module-file fetch failure inside its own wrap.
func TestClosure_ClassifiesTheFetchFailure(t *testing.T) {
	cause := ociregistry.NewHTTPError(errors.New("no such tag"), 404, nil, nil)
	_, err := Closure(context.Background(), errSource{err: cause}, []Dep{{Path: CorePath, Version: "v2.0.0"}})
	require.Error(t, err)
	var fe *oerrors.FetchError
	require.True(t, errors.As(err, &fe))
	assert.Equal(t, oerrors.FetchNotFound, fe.Kind)
	assert.Equal(t, 404, fe.Status)
	assert.NotErrorIs(t, err, oerrors.ErrTransient)
	assert.Equal(t, "resolving dependency "+module.MustNewVersion(CorePath, "v2.0.0").String()+": "+cause.Error(), err.Error())
}
