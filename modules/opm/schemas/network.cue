package schemas

import (
	"strings"
)

/////////////////////////////////////////////////////////////////
//// Network Schemas
/////////////////////////////////////////////////////////////////

// Must start with lowercase letter [a–z],
// end with lowercase letter or digit [a–z0–9],
// and may include hyphens in between.
#IANA_SVC_NAME: string & strings.MinRunes(1) & strings.MaxRunes(15) & =~"^[a-z]([-a-z0-9]{0,13}[a-z0-9])?$"

// Port specification
#PortSchema: {
	// This must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name.
	// Name for the port that can be referred to by services.
	name!: #IANA_SVC_NAME
	// The port that the container will bind to.
	// This must be a valid port number, 0 < x < 65536.
	// If exposedPort is not specified, this value will be used for exposing the port outside the container.
	targetPort!: uint & >=1 & <=65535
	// Protocol for port. Must be UDP, TCP, or SCTP. Defaults to "TCP".
	protocol: *"TCP" | "UDP" | "SCTP"
	// What host IP to bind the external port to.
	hostIP?: string
	// What port to expose on the host.
	// This must be a valid port number, 0 < x < 65536.
	hostPort?: uint & >=1 & <=65535
	// The port that will be exposed outside the container.
	// exposedPort in combination with exposed must inform the platform of what port to map to the container when exposing.
	// This must be a valid port number, 0 < x < 65536.
	exposedPort?: uint & >=1 & <=65535
}

// Expose specification
#ExposeSchema: {
	ports: [portName=string]: #PortSchema & {name: portName}
	type: "ClusterIP" | "NodePort" | "LoadBalancer" | *"ClusterIP"
}

//////////////////////////////////////////////////////////////////
//// Network Rules Schema (for Network Policies)
//////////////////////////////////////////////////////////////////

#NetworkRuleSchema: {
	ingress?: [...{
		from!: [...] // Component references - keeping flexible for now
		ports?: [...#PortSchema]
	}]
	egress?: [...{
		to!: [...] // Component references - keeping flexible for now
		ports?: [...#PortSchema]
	}]
	denyAll?: bool | *false
}

//////////////////////////////////////////////////////////////////
//// Shared Network Schema
//////////////////////////////////////////////////////////////////

#SharedNetworkSchema: {
	networkConfig: {
		dnsPolicy: *"ClusterFirst" | "Default" | "None"
		dnsConfig?: {
			nameservers: [...string]
			searches: [...string]
			options: [...{
				name:   string
				value?: int
			}]
		}
	}
}

//////////////////////////////////////////////////////////////////
//// Route Shared Base Schemas
//////////////////////////////////////////////////////////////////

// Header match for route rules
#RouteHeaderMatch: {
	name!:  string
	value!: string
}

// Base fields shared by all route rules
#RouteRuleBase: {
	backendPort!: uint & >=1 & <=65535
	...
}

// Shared attachment fields for route schemas (gateway, TLS, className)
#RouteAttachmentSchema: {
	gatewayRef?: {
		name!:      string
		namespace?: string
	}
	tls?: {
		mode?: *"Terminate" | "Passthrough"
		certificateRef?: {
			name!:      string
			namespace?: string
		}
	}
	...
}

//////////////////////////////////////////////////////////////////
//// HTTP Route Schemas
//////////////////////////////////////////////////////////////////

// Match criteria for an HTTP route rule
#HttpRouteMatchSchema: {
	path?: {
		type:   *"PathPrefix" | "Exact" | "RegularExpression"
		value!: string
	}
	headers?: [...#RouteHeaderMatch]
	method?: "GET" | "POST" | "PUT" | "DELETE" | "PATCH" | "HEAD" | "OPTIONS"
}

// A single HTTP route rule (embeds RouteRuleBase)
#HttpRouteRuleSchema: #RouteRuleBase & {
	matches?: [...#HttpRouteMatchSchema]
}

// HTTP route specification (embeds RouteAttachmentSchema)
#HttpRouteSchema: #RouteAttachmentSchema & {
	hostnames?: [...string]
	rules: [#HttpRouteRuleSchema, ...#HttpRouteRuleSchema]
}

//////////////////////////////////////////////////////////////////
//// gRPC Route Schemas
//////////////////////////////////////////////////////////////////

// Match criteria for a gRPC route rule
#GrpcRouteMatchSchema: {
	service?: string
	method?:  string
	headers?: [...#RouteHeaderMatch]
}

// A single gRPC route rule (embeds RouteRuleBase)
#GrpcRouteRuleSchema: #RouteRuleBase & {
	matches?: [...#GrpcRouteMatchSchema]
}

// gRPC route specification (embeds RouteAttachmentSchema)
#GrpcRouteSchema: #RouteAttachmentSchema & {
	hostnames?: [...string]
	rules: [#GrpcRouteRuleSchema, ...#GrpcRouteRuleSchema]
}

//////////////////////////////////////////////////////////////////
//// TCP Route Schemas
//////////////////////////////////////////////////////////////////

// A single TCP route rule (embeds RouteRuleBase, no L7 match fields)
#TcpRouteRuleSchema: #RouteRuleBase

// TCP route specification (embeds RouteAttachmentSchema)
#TcpRouteSchema: #RouteAttachmentSchema & {
	rules: [#TcpRouteRuleSchema, ...#TcpRouteRuleSchema]
}

//////////////////////////////////////////////////////////////////
//// TLS Route Schemas
//////////////////////////////////////////////////////////////////

// A single TLS route rule (embeds RouteRuleBase, no L7 match fields)
#TlsRouteRuleSchema: #RouteRuleBase

// TLS route specification (embeds RouteAttachmentSchema)
#TlsRouteSchema: #RouteAttachmentSchema & {
	hostnames?: [...string]
	rules: [#TlsRouteRuleSchema, ...#TlsRouteRuleSchema]
}
