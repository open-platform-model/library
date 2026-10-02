## Context

`example.com/catalogs/extra@v1` replaces the retired path wherever a test needs a second catalog
only as a sample. Generated aliases are positional over the sorted path order, and `example.com`
sorts before `opmodel.dev`, so every `cat0`/`cat1` assignment and sorted-order golden in
`generate_test.go` is unchanged. The test's bare version `1.0.0-alpha.2` is arbitrary and stays.

`.cascade-frozen` exists so release-cascade tooling never rewrites a version literal of an
OPM-owned path in a listed file. `example.com/catalogs/extra@v1` is not OPM-owned, so the
cascade never touches it and it needs no entry; the two files keep their core and
`catalogs/opm` pins.
