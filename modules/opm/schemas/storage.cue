package schemas

/////////////////////////////////////////////////////////////////
//// Volume Schemas
/////////////////////////////////////////////////////////////////

// Volume mount specification - defines container mount point
#VolumeMountSchema: {
	#VolumeSchema

	mountPath!: string
	subPath?:   string
	readOnly:   bool | *false
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
	optional?:    bool | *false
}

// Volume specification - defines storage source
#VolumeSchema: {
	name!: string

	// Only one of these can be set - defines the type of volume
	emptyDir?:        #EmptyDirSchema
	persistentClaim?: #PersistentClaimSchema
	configMap?:       #ConfigMapSchema
	secret?:          #SecretVolumeSourceSchema
	hostPath?:        #HostPathSchema
	nfs?:             #NFSVolumeSourceSchema

	// Exactly one volume source must be set
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
	readOnly:   bool | *false
}

// EmptyDir specification
#EmptyDirSchema: {
	medium?:    *"node" | "memory"
	sizeLimit?: string
}

// HostPath specification - mounts a file or directory from the host node
#HostPathSchema: {
	path!: string
	type?: *"" | "DirectoryOrCreate" | "Directory" | "FileOrCreate" | "File" | "Socket" | "CharDevice" | "BlockDevice"
}

// NFS volume source - mounts a directory from an NFS server
#NFSVolumeSourceSchema: {
	server!:   string // NFS server hostname or IP (e.g. "10.10.0.2")
	path!:     string // Exported NFS path (e.g. "/mnt/data/minecraft")
	readOnly?: bool
}

// Persistent claim specification. To mount a CIFS/SMB share, use a storageClass
// that matches a pre-installed SMB StorageClass (e.g. storageClass: "smb") and set
// accessMode: "ReadWriteMany". The StorageClass and credentials Secret must be
// pre-provisioned on the cluster (see the smb.csi.k8s.io CSI driver for setup).
#PersistentClaimSchema: {
	size:         string
	accessMode:   "ReadWriteOnce" | "ReadOnlyMany" | "ReadWriteMany" | *"ReadWriteOnce"
	storageClass: string | *"standard"
}
