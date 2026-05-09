package schemas

import (
	"math"
	"strconv"
	"strings"
)

// #NormalizeCPU normalizes CPU input to Kubernetes canonical form.
//   number: interpreted as whole/fractional cores (2 → "2", 0.5 → "500m", 2.5 → "2500m")
//   string: millicore format, normalized to canonical form ("2000m" → "2", "500m" → "500m")
//
// Note: "in" is a regular (non-hidden) field so that if-guards evaluate correctly
// when the definition is unified from outside this package.
#NormalizeCPU: {
	in:  number | string
	out: string
	if (in & number) != _|_ {
		let m = math.Round(in * 1000)
		if math.Mod(m, 1000) == 0 {
			out: strconv.FormatInt(m div 1000, 10)
		}
		if math.Mod(m, 1000) != 0 {
			out: strconv.FormatInt(m, 10) + "m"
		}
	}
	if (in & string) != _|_ {
		let m = strconv.Atoi(strings.TrimSuffix(in & =~"^[0-9]+m$", "m"))
		if math.Mod(m, 1000) == 0 {
			out: strconv.FormatInt(m div 1000, 10)
		}
		if math.Mod(m, 1000) != 0 {
			out: strconv.FormatInt(m, 10) + "m"
		}
	}
}

// #NormalizeMemory normalizes memory input to Kubernetes binary format.
//   number: interpreted as GiB (4 → "4Gi", 0.5 → "512Mi")
//   string: passthrough, must match Mi/Gi format ("256Mi", "4Gi")
//
// Note: "in" is a regular (non-hidden) field so that if-guards evaluate correctly
// when the definition is unified from outside this package.
#NormalizeMemory: {
	in:  number | string
	out: string & =~"^[0-9]+[MG]i$"
	if (in & number) != _|_ {
		if math.Remainder(in, 1) == 0 {
			out: "\(math.Round(in))Gi"
		}
		if math.Remainder(in, 1) != 0 {
			out: "\(math.Round(in*1024))Mi"
		}
	}
	if (in & string) != _|_ {
		out: in & =~"^[0-9]+[MG]i$"
	}
}
