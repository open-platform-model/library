package loader

import (
	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/apiversion"
	"github.com/open-platform-model/library/pkg/helper/loader/file"
)

// LoadOptions is a deprecation alias for [file.LoadOptions].
//
// Deprecated: use [file.LoadOptions] from
// github.com/open-platform-model/library/pkg/helper/loader/file.
type LoadOptions = file.LoadOptions

// LoadModulePackage delegates to [file.LoadModulePackage].
//
// Deprecated: use [file.LoadModulePackage] from
// github.com/open-platform-model/library/pkg/helper/loader/file.
func LoadModulePackage(ctx *cue.Context, dirPath string) (cue.Value, apiversion.Version, error) {
	return file.LoadModulePackage(ctx, dirPath)
}

// LoadReleaseFile delegates to [file.LoadReleaseFile].
//
// Deprecated: use [file.LoadReleaseFile] from
// github.com/open-platform-model/library/pkg/helper/loader/file.
func LoadReleaseFile(ctx *cue.Context, filePath string, opts LoadOptions) (cue.Value, string, apiversion.Version, error) {
	return file.LoadReleaseFile(ctx, filePath, opts)
}

// LoadValuesFile delegates to [file.LoadValuesFile].
//
// Deprecated: use [file.LoadValuesFile] from
// github.com/open-platform-model/library/pkg/helper/loader/file.
func LoadValuesFile(ctx *cue.Context, path string) (cue.Value, error) {
	return file.LoadValuesFile(ctx, path)
}
