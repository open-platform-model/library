// Kubernetes networking/v1 API group re-exports
package v1

import netv1 "cue.dev/x/k8s.io/api/networking/v1"

#HTTPIngressPath:                 netv1.#HTTPIngressPath
#HTTPIngressRuleValue:            netv1.#HTTPIngressRuleValue
#Ingress:                         netv1.#Ingress
#IngressBackend:                  netv1.#IngressBackend
#IngressClass:                    netv1.#IngressClass
#IngressClassList:                netv1.#IngressClassList
#IngressClassParametersReference: netv1.#IngressClassParametersReference
#IngressClassSpec:                netv1.#IngressClassSpec
#IngressList:                     netv1.#IngressList
#IngressLoadBalancerIngress:      netv1.#IngressLoadBalancerIngress
#IngressLoadBalancerStatus:       netv1.#IngressLoadBalancerStatus
#IngressPortStatus:               netv1.#IngressPortStatus
#IngressRule:                     netv1.#IngressRule
#IngressServiceBackend:           netv1.#IngressServiceBackend
#IngressSpec:                     netv1.#IngressSpec
#IngressStatus:                   netv1.#IngressStatus
#IngressTLS:                      netv1.#IngressTLS
#IPAddress:                       netv1.#IPAddress
#IPAddressList:                   netv1.#IPAddressList
#IPAddressSpec:                   netv1.#IPAddressSpec
#IPBlock:                         netv1.#IPBlock
#NetworkPolicy:                   netv1.#NetworkPolicy
#NetworkPolicyEgressRule:         netv1.#NetworkPolicyEgressRule
#NetworkPolicyIngressRule:        netv1.#NetworkPolicyIngressRule
#NetworkPolicyList:               netv1.#NetworkPolicyList
#NetworkPolicyPeer:               netv1.#NetworkPolicyPeer
#NetworkPolicyPort:               netv1.#NetworkPolicyPort
#NetworkPolicySpec:               netv1.#NetworkPolicySpec
#ParentReference:                 netv1.#ParentReference
#ServiceBackendPort:              netv1.#ServiceBackendPort
#ServiceCIDR:                     netv1.#ServiceCIDR
#ServiceCIDRList:                 netv1.#ServiceCIDRList
#ServiceCIDRSpec:                 netv1.#ServiceCIDRSpec
#ServiceCIDRStatus:               netv1.#ServiceCIDRStatus
