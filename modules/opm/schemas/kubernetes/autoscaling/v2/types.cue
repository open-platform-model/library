// Kubernetes autoscaling/v2 API group re-exports
package v2

import asv2 "cue.dev/x/k8s.io/api/autoscaling/v2"

#ContainerResourceMetricSource:    asv2.#ContainerResourceMetricSource
#ContainerResourceMetricStatus:    asv2.#ContainerResourceMetricStatus
#CrossVersionObjectReference:      asv2.#CrossVersionObjectReference
#ExternalMetricSource:             asv2.#ExternalMetricSource
#ExternalMetricStatus:             asv2.#ExternalMetricStatus
#HorizontalPodAutoscaler:          asv2.#HorizontalPodAutoscaler
#HorizontalPodAutoscalerBehavior:  asv2.#HorizontalPodAutoscalerBehavior
#HorizontalPodAutoscalerCondition: asv2.#HorizontalPodAutoscalerCondition
#HorizontalPodAutoscalerList:      asv2.#HorizontalPodAutoscalerList
#HorizontalPodAutoscalerSpec:      asv2.#HorizontalPodAutoscalerSpec
#HorizontalPodAutoscalerStatus:    asv2.#HorizontalPodAutoscalerStatus
#HPAScalingPolicy:                 asv2.#HPAScalingPolicy
#HPAScalingRules:                  asv2.#HPAScalingRules
#MetricIdentifier:                 asv2.#MetricIdentifier
#MetricSpec:                       asv2.#MetricSpec
#MetricStatus:                     asv2.#MetricStatus
#MetricTarget:                     asv2.#MetricTarget
#MetricValueStatus:                asv2.#MetricValueStatus
#ObjectMetricSource:               asv2.#ObjectMetricSource
#ObjectMetricStatus:               asv2.#ObjectMetricStatus
#PodsMetricSource:                 asv2.#PodsMetricSource
#PodsMetricStatus:                 asv2.#PodsMetricStatus
#ResourceMetricSource:             asv2.#ResourceMetricSource
#ResourceMetricStatus:             asv2.#ResourceMetricStatus
