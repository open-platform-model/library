package v1alpha2

// Copied from enhancement 004 / 03-schema.md (post-slim). Identity-only.
// No #platform / #environment inputs; #ComponentNames self-defaults its
// _clusterDomain to "cluster.local" with no override path.

#ModuleContext: {
	runtime: #RuntimeContext
}

#RuntimeContext: {
	release: {
		name!:      #NameType
		namespace!: string
		uuid!:      #UUIDType
	}

	module: {
		name!:    #NameType
		version!: #VersionType
		fqn!:     #ModuleFQNType
		uuid!:    #UUIDType
	}

	components: [compName=string]: #ComponentNames & {
		_releaseName: release.name
		_namespace:   release.namespace
		_compName:    compName
	}
}

#ComponentNames: {
	_releaseName:   string
	_namespace:     string
	_compName:      string
	_clusterDomain: string | *"cluster.local"

	resourceName: string | *"\(_releaseName)-\(_compName)"

	dns: {
		local:      resourceName
		namespaced: "\(resourceName).\(_namespace)"
		svc:        "\(resourceName).\(_namespace).svc"
		fqdn:       "\(resourceName).\(_namespace).svc.\(_clusterDomain)"
	}
}
