package storage

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
)

#VolumesResource: c.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/resources/storage"
		version:     "v1"
		name:        "volumes"
		description: "A volume definition for workloads"
		labels: {
			"resource.opmodel.dev/category": "storage"
		}
	}

	spec: volumes: [volumeName=string]: schemas.#VolumeSchema & {name: string | *volumeName}
}
