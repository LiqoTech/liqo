// +build !ignore_autogenerated

/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForeignCluster) DeepCopyInto(out *ForeignCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForeignCluster.
func (in *ForeignCluster) DeepCopy() *ForeignCluster {
	if in == nil {
		return nil
	}
	out := new(ForeignCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ForeignCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForeignClusterList) DeepCopyInto(out *ForeignClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ForeignCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForeignClusterList.
func (in *ForeignClusterList) DeepCopy() *ForeignClusterList {
	if in == nil {
		return nil
	}
	out := new(ForeignClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ForeignClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForeignClusterSpec) DeepCopyInto(out *ForeignClusterSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForeignClusterSpec.
func (in *ForeignClusterSpec) DeepCopy() *ForeignClusterSpec {
	if in == nil {
		return nil
	}
	out := new(ForeignClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForeignClusterStatus) DeepCopyInto(out *ForeignClusterStatus) {
	*out = *in
	in.Outgoing.DeepCopyInto(&out.Outgoing)
	in.Incoming.DeepCopyInto(&out.Incoming)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForeignClusterStatus.
func (in *ForeignClusterStatus) DeepCopy() *ForeignClusterStatus {
	if in == nil {
		return nil
	}
	out := new(ForeignClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Incoming) DeepCopyInto(out *Incoming) {
	*out = *in
	if in.PeeringRequest != nil {
		in, out := &in.PeeringRequest, &out.PeeringRequest
		*out = new(corev1.ObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Incoming.
func (in *Incoming) DeepCopy() *Incoming {
	if in == nil {
		return nil
	}
	out := new(Incoming)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OriginClusterSets) DeepCopyInto(out *OriginClusterSets) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OriginClusterSets.
func (in *OriginClusterSets) DeepCopy() *OriginClusterSets {
	if in == nil {
		return nil
	}
	out := new(OriginClusterSets)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Outgoing) DeepCopyInto(out *Outgoing) {
	*out = *in
	if in.CaDataRef != nil {
		in, out := &in.CaDataRef, &out.CaDataRef
		*out = new(corev1.ObjectReference)
		**out = **in
	}
	if in.Advertisement != nil {
		in, out := &in.Advertisement, &out.Advertisement
		*out = new(corev1.ObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Outgoing.
func (in *Outgoing) DeepCopy() *Outgoing {
	if in == nil {
		return nil
	}
	out := new(Outgoing)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PeeringRequest) DeepCopyInto(out *PeeringRequest) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PeeringRequest.
func (in *PeeringRequest) DeepCopy() *PeeringRequest {
	if in == nil {
		return nil
	}
	out := new(PeeringRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PeeringRequest) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PeeringRequestList) DeepCopyInto(out *PeeringRequestList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PeeringRequest, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PeeringRequestList.
func (in *PeeringRequestList) DeepCopy() *PeeringRequestList {
	if in == nil {
		return nil
	}
	out := new(PeeringRequestList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PeeringRequestList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PeeringRequestSpec) DeepCopyInto(out *PeeringRequestSpec) {
	*out = *in
	if in.KubeConfigRef != nil {
		in, out := &in.KubeConfigRef, &out.KubeConfigRef
		*out = new(corev1.ObjectReference)
		**out = **in
	}
	out.OriginClusterSets = in.OriginClusterSets
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PeeringRequestSpec.
func (in *PeeringRequestSpec) DeepCopy() *PeeringRequestSpec {
	if in == nil {
		return nil
	}
	out := new(PeeringRequestSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PeeringRequestStatus) DeepCopyInto(out *PeeringRequestStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PeeringRequestStatus.
func (in *PeeringRequestStatus) DeepCopy() *PeeringRequestStatus {
	if in == nil {
		return nil
	}
	out := new(PeeringRequestStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SearchDomain) DeepCopyInto(out *SearchDomain) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SearchDomain.
func (in *SearchDomain) DeepCopy() *SearchDomain {
	if in == nil {
		return nil
	}
	out := new(SearchDomain)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SearchDomain) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SearchDomainList) DeepCopyInto(out *SearchDomainList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]SearchDomain, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SearchDomainList.
func (in *SearchDomainList) DeepCopy() *SearchDomainList {
	if in == nil {
		return nil
	}
	out := new(SearchDomainList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SearchDomainList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SearchDomainSpec) DeepCopyInto(out *SearchDomainSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SearchDomainSpec.
func (in *SearchDomainSpec) DeepCopy() *SearchDomainSpec {
	if in == nil {
		return nil
	}
	out := new(SearchDomainSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SearchDomainStatus) DeepCopyInto(out *SearchDomainStatus) {
	*out = *in
	if in.ForeignClusters != nil {
		in, out := &in.ForeignClusters, &out.ForeignClusters
		*out = make([]corev1.ObjectReference, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SearchDomainStatus.
func (in *SearchDomainStatus) DeepCopy() *SearchDomainStatus {
	if in == nil {
		return nil
	}
	out := new(SearchDomainStatus)
	in.DeepCopyInto(out)
	return out
}
