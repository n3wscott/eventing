# Object Model

Knative ships with several objects which implement useful sets of
[interfaces](interfaces.md). It is expected that additional objects which
implement other sets of interfaces will be added over time. For context, see
the [overview](overview.md) and [motivations](motivation.md) sections.

These are Kubernetes resources that been introduced using Custom Resource
Definitions. They will have the expected _ObjectMeta_, _Spec_, _Status_ fields.
This document details our _Spec_ and _Status_ customizations.

- [Source](#kind-source)
- [Channel](#kind-channel)
- [Subscription](#kind-subscription)
- [Provider](#kind-provisioner)

---

## kind: Source

### group: eventing.knative.dev/v1alpha1

_Describes a specific configuration (credentials, etc) of a source system which
can be used to supply events. A common pattern is for Sources to emit events to
Channel to allow event delivery to be fanned-out within the cluster. They
cannot receive events._

### Object Schema

#### Spec

| Field         | Type                               | Description                                                                                                            | Limitations                                               |
| ------------- | ---------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| provisioner\* | ProvisionerReference               | The provisioner used to create any backing resources and configuration.                                                | Immutable.                                                |
| arguments     | runtime.RawExtension (JSON object) | Arguments passed to the provisioner for this specific source.                                                          | Arguments must validate against provisioner's parameters. |
| channel\*     | ObjectRef                          | Specify a Channel to target.                                                                                           | Source will not emit events until channel exists.         |

\*: Required

#### Status

| Field        | Type                      | Description                                                                                  | Limitations                                              |
| ------------ | ------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| provisioned  | []ProvisionedObjectStatus | Creation status of each Channel and errors therein.                                          | It is expected that a Source list all produced Channels. |
| conditions   | Conditions                | Source conditions.                                                                           |                                                          |

##### Conditions

- **Ready.** True when the Source is provisioned and ready to emit events.
- **Provisioned.** True when the Source has been provisioned by a controller.

#### Events

- Provisioned - describes each resource that is provisioned.

### Life Cycle

| Action | Reactions                                                                                                 | Limitations |
| ------ | --------------------------------------------------------------------------------------------------------- | ----------- |
| Create | Provisioner controller watches for Sources and creates the backing resources depending on implementation. |             |
| Update | Provisioner controller synchronizes backing implementation on changes.                                    |             |
| Delete | Provisioner controller will deprovision backing resources depending on implementation.                    |             |

---

## kind: Channel

### group: eventing.knative.dev/v1alpha1

_A Channel logically receives events on its input domain and forwards them to
its subscribers. Additional behavior may be introduced by using the
Subscription's call parameter._

### Object Schema

#### Spec

| Field         | Type                               | Description                                                                | Limitations                          |
| ------------- | ---------------------------------- | -------------------------------------------------------------------------- | ------------------------------------ |
| provisioner\* | ProvisionerReference               | The name of the provisioner to create the resources that back the Channel. | Immutable.                           |
| arguments     | runtime.RawExtension (JSON object) | Arguments to be passed to the provisioner.                                 |                                      |
| channelable   | Channelable                        | Holds a list of downstream subscribers for the channel.                    |                                      |
| eventTypes    | []String                           | An array of event types that will be passed on the Channel.                | Must be objects with kind:EventType. |

\*: Required

#### Metadata

##### Owner References

- If the Source controller created this Channel: Owned by the originating
  Source.
- Owned (non-controlling) by the Provisioner used to provision the Channel.

#### Status

| Field        | Type         | Description                                                                                                                 | Limitations |
| ------------ | ------------ | --------------------------------------------------------------------------------------------------------------------------- | ----------- |
| sinkable     | Sinkable     | Address to the endpoint as top-level domain that will distribute traffic over the provided targets from inside the cluster. |             |
| conditions   | Conditions   | Standard Subscriptions                                                                                                      |             |

##### Conditions

- **Ready.** True when the Channel is provisioned and ready to accept events.
- **Provisioned.** True when the Channel has been provisioned by a controller.

#### Events

- Provisioned
- Deprovisioned

### Life Cycle

| Action | Reactions                                                                                                                                                        | Limitations                                                          |
| ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| Create | The Provisioner referenced will take ownership of the Channel and begin provisioning the backing resources required for the Channel depending on implementation. | Only one Provisioner is allowed to be the Owner for a given Channel. |
| Update | The Provisioner will synchronize the Channel backing resources to reflect the update.                                                                            |                                                                      |
| Delete | The Provisioner will deprovision the backing resources if no longer required depending on implementation.                                                        |                                                                      |

---

## kind: Subscription

### group: eventing.knative.dev/v1alpha1

_Describes a linkage between a Channel and a Targetable and/or Sinkable._

### Object Schema

#### Spec

| Field              | Type         | Description                                                                  | Limitations        |
| ------------------ | ------------ | ---------------------------------------------------------------------------- | ------------------ |
| from\*             | ObjectRef    | The originating Channel for the link.                                        | Must be a Channel. |
| call<sup>1</sup>   | EndpointSpec | Optional processing on the event. The result of call will be sent to result. |                    |
| result<sup>1</sup> | ObjectRef    | The continuation Channel for the link.                                       | Must be a Channel. |

\*: Required

1: At Least One(call, result)

#### Metadata

##### Owner References

- If a resource controller created this Subscription: Owned by the originating
  resource.

##### Conditions

- **Ready.**
- **FromReady.**
- **CallActive.** True if the call is sinking events without error.
- **Resolved.**

#### Events

- PublisherAcknowledged
- ActionFailed

### Life Cycle

There are two phases for Subscription reconciliation:

1. resolving DNS names for the Subscription into a ResolvedSubscription
   - create a new ResolvedSubscription
   - set _role_ as the resolved 'eventing.knative.dev/role' annotation for the _spec.call_ target resource, or default to 'transformer' if not set
   - set _callableDomain_ as the resolved DNS name for the _spec.call_ target resource according to the Targetable interface
   - set _sinkableDomain_ as the resolved DNS name for the _spec.result_ target resource according to the Sinkable interface
   - add an entry to the _routeMapping_ map for each ResultStrategyRoute setting the key from the _name_ and the value as the resolved DNS name for the _spec.result_ target resource according to the Sinkable interface
2. updating the ResolvedSubscriptionSet resource
   - resolve the resource referenced in the Subscription's _spec.from_
   - from that resource, resolve the ResolvedSubscriptionSet via the Subscribable interface
   - add/update/remove the ResolvedSubscription in the ResolvedSubscriptionSet resource

| Action | Reactions                                                                                                                                                                                               | Limitations |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- |
| Create | Create a ResolvedSubscription entry for the Subscription, add the new entry to the ResolvedSubscriptionSet specified by the Subscribable interface on the resource referenced by the _spec.from_ field. |             |
| Update | Update the ResolvedSubscription entry for the Subscription with updated DNS name.                                                                                                                       |             |
| Delete | Remove the ResolvedSubscription entry for the Subscription.                                                                                                                                             |             |

---

## kind: ResolvedSubscriptionSet

### group: eventing.internal.knative.dev/v1alpha1

_A ResolvedSubscriptionSet holds an aggregation of resolved subscriptions for a
Channel._

### Object Schema

#### Spec

| Field         | Type                   | Description                                                           | Limitations                            |
| ------------- | ---------------------- | --------------------------------------------------------------------- | -------------------------------------- |
| subscribers\* | ResolvedSubscription[] | Information about subscriptions used to implement message forwarding. | Filled out by Subscription Controller. |

\*: Required

#### Metadata

##### Owner References

- Owned (controlling) by the resource the subscribers are for.

---

## kind: Provisioner

### group: eventing.knative.dev/v1alpha1

_Describes an abstract configuration of a Source system which produces events
or a Channel system that receives and delivers events._

### Object Schema

#### Spec

| Field  | Type                                                                            | Description                                 | Limitations                |
| ------ | ------------------------------------------------------------------------------- | ------------------------------------------- | -------------------------- |
| type\* | [GroupKind](https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema#GroupKind) | The type of the resource to be provisioned. | Must be Source or Channel. |

\*: Required

#### Metadata

##### Owner References

- Owns EventTypes.

#### Status

| Field       | Type                      | Description                                                          | Limitations                                                                    |
| ----------- | ------------------------- | -------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| provisioned | []ProvisionedObjectStatus | Status of creation or adoption of each EventType and errors therein. | It is expected that a provisioner list all produced EventTypes, if applicable. |
| conditions  | Conditions                | Provisioner conditions                                               |                                                                                |

##### Conditions

- **Ready.**

#### Events

- Source created
- Source deleted
- Event types installed

### Life Cycle

| Action | Reactions                                                                       | Limitations                                                                                              |
| ------ | ------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| Create | Creates and owns EventTypes produced, or adds Owner ref to existing EventTypes. | Verifies Json Schema provided by existing EventTypes; Not allowed to edit EventType if previously Owned; |
| Update | Synchronizes EventTypes.                                                        |                                                                                                          |
| Delete | Removes Owner ref from EventTypes.                                              |                                                                                                          |

---

## kind: EventType

NOTE: EventType is out of scope for 0.1 release. This is future documentation.

### group: eventing.knative.dev/v1alpha1

_Describes a particular schema of Events which may be produced by one or more
source systems._

### Object Schema

#### Spec

| Field      | Type   | Description                                         | Limitations                    |
| ---------- | ------ | --------------------------------------------------- | ------------------------------ |
| jsonSchema | String | The Json Schema that represents an event in flight. | Only for JSON transport types. |

#### Metadata

##### Owner References

EventType is owned by _Provisioners_. Each _Provisioner_ creates a
non-controlling OwnerReference on the EventType resources it knows about.

#### Status

| Field          | Type    | Description                         | Limitations |
| -------------- | ------- | ----------------------------------- | ----------- |
| referenceCount | Integer | Number of Owners for this EventType |             |

#### Events

- **Owned.** When EventType has a new Provisioner Owner.
- **Released.** When a Provisioner removes Ownership from the EventType.

### Life Cycle

| Action | Reactions | Limitations                                |
| ------ | --------- | ------------------------------------------ |
| Create |           |                                            |
| Update |           |                                            |
| Delete |           | Blocked until all provisioners release it. |

---

## Shared Object Schema

### ProvisionerReference

| Field                | Type            | Description | Limitations                   |
| -------------------- | --------------- | ----------- | ----------------------------- |
| ref<sup>1</sup>      | ObjectReference |             |                               |
| selector<sup>1</sup> | LabelSelector   |             | Must match only one resource. |

1: One of (name, selector), Required.

### EndpointSpec

| Field                 | Type            | Description | Limitations                |
| --------------------- | --------------- | ----------- | -------------------------- |
| targetRef<sup>1</sup> | ObjectReference |             | Must adhere to Targetable. |
| dnsName<sup>1</sup>   | String          |             |                            |

1: One of (targetRef, dnsName), Required.

### ProvisionedObjectStatus

| Field    | Type   | Description                                                  | Limitations |
| -------- | ------ | ------------------------------------------------------------ | ----------- |
| name\*   | String | Name of Object                                               |             |
| type\*   | String | Fully Qualified Object type.                                 |             |
| status\* | String | Current relationship between Provisioner and Object.         |             |
| reason   | String | Detailed description describing current relationship status. |             |

\*: Required

### Channelable

| Field       | Type                    | Description                                                           | Limitations                            |
| ----------- | ----------------------- | --------------------------------------------------------------------- | -------------------------------------- |
| subscribers | ChannelSubscriberSpec[] | Information about subscriptions used to implement message forwarding. | Filled out by Subscription Controller. |

### ChannelSubscriberSpec

| Field          | Type              | Description                                                       | Limitations                                               |
| -------------- | ----------------- | ----------------------------------------------------------------- | --------------------------------------------------------- |
| role           | String            | How to forward the response from callableDomain to sinkableDomain | One of: transformer, filter, router, splitter, aggregator |
| callableDomain | String            | The domain name of the endpoint for the call.                     |                                                           |
| sinkableDomain | String            | The domain name of the endpoint for the result.                   |                                                           |
| routeMapping   | map[String]String | Named routes for the router role.                                 | The role must be router                                   |

### ResultStrategy

| Field                    | Type                  | Description                            | Limitations                         |
| ------------------------ | --------------------- | -------------------------------------- | ----------------------------------- |
| target<sup>1</sup>       | ObjectRef             | The continuation Channel for the link. | Must be a Channel.                  |
| routeMapping<sup>1</sup> | []ResultStrategyRoute | Named routes for the router role       | The \_\_ must be of the router role |

1: At least one of (target, routeMapping), Required.

### ResultStrategyRoute

| Field  | Type      | Description                                    | Limitations        |
| ------ | --------- | ---------------------------------------------- | ------------------ |
| name   | string    | The name retruned from the 'router' component. |                    |
| target | ObjectRef | The continuation Channel for the link.         | Must be a Channel. |
