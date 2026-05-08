// Doc-only file. The package overview lives on api.go.
//
// # Registration contract
//
// Each per-version binding package (e.g. pkg/api/v1alpha2) MUST register
// itself from init():
//
//	package v1alpha2
//
//	import "github.com/open-platform-model/library/pkg/api"
//
//	func init() { api.Register(&binding{}) }
//
// Consumers of the kernel pull a binding into their build by importing the
// per-version package (typically with a blank import in main):
//
//	import _ "github.com/open-platform-model/library/pkg/api/v1alpha2"
//
// # Why init()
//
// Multiple binaries consume the kernel (CLI, controller, future Crossplane
// composition function). Relying on each binary to call api.Register from
// main() is a footgun: forgetting it produces a runtime ErrUnknownAPIVersion
// instead of a compile-time error. init() registration matches the standard
// library plug-in pattern (database/sql, image, encoding/json subtypes).
//
// # Why panic on duplicate
//
// Two packages registering the same apiversion.Version is a programming error,
// not a recoverable runtime condition. Panicking from Register surfaces the
// misconfiguration at process start (during the init() chain), which is the
// earliest point at which any caller could observe the inconsistency. This
// matches sql.Register's behaviour and keeps the registry deterministic for
// every later read.
package api
