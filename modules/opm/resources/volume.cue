package resources

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
)

/////////////////////////////////////////////////////////////////
//// Volumes Resource
/////////////////////////////////////////////////////////////////

#VolumesResource: c.#Resource & {
	metadata: {
		modulePath:  "\(id.ModulePath)/resources"
		version:     id.Version
		name:        "volumes"
		description: "A volume definition for workloads"
		labels: {
			"resource.opmodel.dev/category": "storage"
		}
	}

	spec: volumes: [volumeName=string]: #VolumeSchema & {name: string | *volumeName}
}

#Volumes: c.#Component & {
	#resources: (#VolumesResource.metadata.fqn): #VolumesResource
}

/////////////////////////////////////////////////////////////////
//// Volume Schemas
/////////////////////////////////////////////////////////////////

// Volume mount spec — defines container mount point. Referenced by #ContainerSchema.
#VolumeMountSchema: {
	#VolumeSchema

	mountPath!: string
	subPath?:   string
	readOnly:   bool
}

#VolumeMountDefaults: #VolumeMountSchema & {
	readOnly: false
}

#FileMode: int & >=0 & <=511

#SecretVolumeItemSchema: {
	key!:  string
	path!: string
	mode?: #FileMode
}

#SecretVolumeSourceSchema: {
	from!: #SecretSchema
	items?: [...#SecretVolumeItemSchema]
	defaultMode?: #FileMode
	optional?:    bool
}

#SecretVolumeSourceDefaults: #SecretVolumeSourceSchema & {
	optional: false
}

// Volume specification — defines storage source. Exactly one source must be set.
#VolumeSchema: {
	name!: string

	emptyDir?:        #EmptyDirSchema
	persistentClaim?: #PersistentClaimSchema
	configMap?:       #ConfigMapSchema
	secret?:          #SecretVolumeSourceSchema
	hostPath?:        #HostPathSchema
	nfs?:             #NFSVolumeSourceSchema

	matchN(1, [
		{emptyDir!: _},
		{persistentClaim!: _},
		{configMap!: _},
		{secret!: _},
		{hostPath!: _},
		{nfs!: _},
	])

	mountPath?: string
	subPath?:   string
	readOnly:   bool
}

#VolumeDefaults: #VolumeSchema & {
	readOnly: false
}

#EmptyDirSchema: {
	medium?:    "node" | "memory"
	sizeLimit?: string
}

#EmptyDirDefaults: #EmptyDirSchema & {
	medium: "node"
}

// Mounts a file or directory from the host node.
#HostPathSchema: {
	path!: string
	type?: "" | "DirectoryOrCreate" | "Directory" | "FileOrCreate" | "File" | "Socket" | "CharDevice" | "BlockDevice"
}

#HostPathDefaults: #HostPathSchema & {
	type: ""
}

// Mounts a directory from an NFS server.
#NFSVolumeSourceSchema: {
	server!:   string
	path!:     string
	readOnly?: bool
}

// To mount a CIFS/SMB share use a storageClass that matches a pre-installed
// SMB StorageClass (e.g. "smb") with accessMode: "ReadWriteMany". The
// StorageClass and credentials Secret must be pre-provisioned (see the
// smb.csi.k8s.io CSI driver).
#PersistentClaimSchema: {
	size:         string
	accessMode:   "ReadWriteOnce" | "ReadOnlyMany" | "ReadWriteMany"
	storageClass: string
}

#PersistentClaimDefaults: #PersistentClaimSchema & {
	accessMode:   "ReadWriteOnce"
	storageClass: "standard"
}
