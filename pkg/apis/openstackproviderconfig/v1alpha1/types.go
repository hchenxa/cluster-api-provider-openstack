/*
Copyright 2018 The Kubernetes Authors.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubeadmv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OpenstackProviderSpec is the type that will be embedded in a Machine.Spec.ProviderSpec field
// for an OpenStack Instance. It is used by the Openstack machine actuator to create a single machine instance.
// TODO(cglaubitz): We might consider to change this to OpenstackMachineProviderSpec
// +k8s:openapi-gen=true
type OpenstackProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The name of the secret containing the openstack credentials
	CloudsSecret *corev1.SecretReference `json:"cloudsSecret"`

	// The name of the cloud to use from the clouds secret
	CloudName string `json:"cloudName"`

	// The flavor reference for the flavor for your server instance.
	Flavor string `json:"flavor"`

	// The name of the image to use for your server instance.
	// If the RootVolume is specified, this will be ignored and use rootVolume directly.
	Image string `json:"image"`

	// The ssh key to inject in the instance
	KeyName string `json:"keyName,omitempty"`

	// A networks object. Required parameter when there are multiple networks defined for the tenant.
	// When you do not specify the networks parameter, the server attaches to the only network created for the current tenant.
	Networks []NetworkParam `json:"networks,omitempty"`
	// The floatingIP which will be associated to the machine, only used for master.
	// The floatingIP should have been created and haven't been associated.
	FloatingIP string `json:"floatingIP,omitempty"`

	// The availability zone from which to launch the server.
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// The names of the security groups to assign to the instance
	SecurityGroups []SecurityGroupParam `json:"securityGroups,omitempty"`

	// The name of the secret containing the user data (startup script in most cases)
	UserDataSecret *corev1.SecretReference `json:"userDataSecret,omitempty"`

	// Whether the server instance is created on a trunk port or not.
	Trunk bool `json:"trunk,omitempty"`

	// Machine tags
	// Requires Nova api 2.52 minimum!
	Tags []string `json:"tags,omitempty"`

	// Metadata mapping. Allows you to create a map of key value pairs to add to the server instance.
	ServerMetadata map[string]string `json:"serverMetadata,omitempty"`

	// Config Drive support
	ConfigDrive *bool `json:"configDrive,omitempty"`

	// The volume metadata to boot from
	RootVolume *RootVolume `json:"rootVolume,omitempty"`

	// KubeadmConfiguration holds the kubeadm configuration options
	// +optional
	KubeadmConfiguration KubeadmConfiguration `json:"kubeadmConfiguration,omitempty"`
}

// KubeadmConfiguration holds the various configurations that kubeadm uses
type KubeadmConfiguration struct {
	// JoinConfiguration is used to customize any kubeadm join configuration
	// parameters.
	Join kubeadmv1beta1.JoinConfiguration `json:"join,omitempty"`

	// InitConfiguration is used to customize any kubeadm init configuration
	// parameters.
	Init kubeadmv1beta1.InitConfiguration `json:"init,omitempty"`
}

type SecurityGroupParam struct {
	// Security Group UID
	UUID string `json:"uuid,omitempty"`
	// Security Group name
	Name string `json:"name,omitempty"`
	// Filters used to query security groups in openstack
	Filter SecurityGroupFilter `json:"filter,omitempty"`
}

type SecurityGroupFilter struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	TenantID    string `json:"tenantId,omitempty"`
	ProjectID   string `json:"projectId,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Marker      string `json:"marker,omitempty"`
	SortKey     string `json:"sortKey,omitempty"`
	SortDir     string `json:"sortDir,omitempty"`
	Tags        string `json:"tags,omitempty"`
	TagsAny     string `json:"tagsAny,omitempty"`
	NotTags     string `json:"notTags,omitempty"`
	NotTagsAny  string `json:"notTagsAny,omitempty"`
}

type NetworkParam struct {
	// The UUID of the network. Required if you omit the port attribute.
	UUID string `json:"uuid,omitempty"`
	// A fixed IPv4 address for the NIC.
	FixedIp string `json:"fixedIp,omitempty"`
	// Filters for optional network query
	Filter Filter `json:"filter,omitempty"`
	// Subnet within a network to use
	Subnets []SubnetParam `json:"subnets,omitempty"`
}

type Filter struct {
	Status       string `json:"status,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	AdminStateUp *bool  `json:"adminStateUp,omitempty"`
	TenantID     string `json:"tenantId,omitempty"`
	ProjectID    string `json:"projectId,omitempty"`
	Shared       *bool  `json:"shared,omitempty"`
	ID           string `json:"id,omitempty"`
	Marker       string `json:"marker,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	SortKey      string `json:"sortKey,omitempty"`
	SortDir      string `json:"sortDir,omitempty"`
	Tags         string `json:"tags,omitempty"`
	TagsAny      string `json:"tagsAny,omitempty"`
	NotTags      string `json:"notTags,omitempty"`
	NotTagsAny   string `json:"notTagsAny,omitempty"`
}

type SubnetParam struct {
	// The UUID of the network. Required if you omit the port attribute.
	UUID string `json:"uuid,omitempty"`

	// Filters for optional network query
	Filter SubnetFilter `json:"filter,omitempty"`
}

type SubnetFilter struct {
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	EnableDHCP      *bool  `json:"enableDhcp,omitempty"`
	NetworkID       string `json:"networkId,omitempty"`
	TenantID        string `json:"tenantId,omitempty"`
	ProjectID       string `json:"projectId,omitempty"`
	IPVersion       int    `json:"ipVersion,omitempty"`
	GatewayIP       string `json:"gateway_ip,omitempty"`
	CIDR            string `json:"cidr,omitempty"`
	IPv6AddressMode string `json:"ipv6AddressMode,omitempty"`
	IPv6RAMode      string `json:"ipv6RaMode,omitempty"`
	ID              string `json:"id,omitempty"`
	SubnetPoolID    string `json:"subnetpoolId,omitempty"`
	Limit           int    `json:"limit,omitempty"`
	Marker          string `json:"marker,omitempty"`
	SortKey         string `json:"sortKey,omitempty"`
	SortDir         string `json:"sortDir,omitempty"`
	Tags            string `json:"tags,omitempty"`
	TagsAny         string `json:"tagsAny,omitempty"`
	NotTags         string `json:"notTags,omitempty"`
	NotTagsAny      string `json:"notTagsAny,omitempty"`
}

type RootVolume struct {
	SourceType string `json:"sourceType,omitempty"`
	SourceUUID string `json:"sourceUUID,omitempty"`
	DeviceType string `json:"deviceType"`
	Size       int    `json:"diskSize,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OpenstackClusterProviderSpec is the providerSpec for OpenStack in the cluster object
// +k8s:openapi-gen=true
type OpenstackClusterProviderSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The name of the secret containing the openstack credentials
	CloudsSecret *corev1.SecretReference `json:"cloudsSecret"`

	// The name of the cloud to use from the clouds secret
	CloudName string `json:"cloudName"`

	// NodeCIDR is the OpenStack Subnet to be created. Cluster actuator will create a
	// network, a subnet with NodeCIDR, and a router connected to this subnet.
	// If you leave this empty, no network will be created.
	NodeCIDR string `json:"nodeCidr,omitempty"`
	// DNSNameservers is the list of nameservers for OpenStack Subnet being created.
	DNSNameservers []string `json:"dnsNameservers,omitempty"`
	// ExternalRouterIPs is an array of externalIPs on the respective subnets.
	// This is necessary if the router needs a fixed ip in a specific subnet.
	ExternalRouterIPs []ExternalRouterIPParam `json:"externalRouterIPs,omitempty"`
	// ExternalNetworkID is the ID of an external OpenStack Network. This is necessary
	// to get public internet to the VMs.
	ExternalNetworkID string `json:"externalNetworkId,omitempty"`
	// UseOctavia is weather LoadBalancer Service is Octavia or not
	UseOctavia bool `json:"useOctavia"`

	// ManagedAPIServerLoadBalancer defines whether a LoadBalancer for the
	// APIServer should be created. If set to true the following properties are
	// mandatory: APIServerLoadBalancerFloatingIP, APIServerLoadBalancerPort
	ManagedAPIServerLoadBalancer bool `json:"managedAPIServerLoadBalancer"`

	// APIServerLoadBalancerFloatingIP is the floatingIP which will be associated
	// to the APIServer loadbalancer. The floatingIP will be created if it not
	// already exists.
	APIServerLoadBalancerFloatingIP string `json:"apiServerLoadBalancerFloatingIP,omitempty"`

	// APIServerLoadBalancerPort is the port on which the listener on the APIServer
	// loadbalancer will be created
	APIServerLoadBalancerPort int `json:"apiServerLoadBalancerPort,omitempty"`

	// APIServerLoadBalancerAdditionalPorts adds additional ports to the APIServerLoadBalancer
	APIServerLoadBalancerAdditionalPorts []int `json:"apiServerLoadBalancerAdditionalPorts,omitempty"`

	// ManagedSecurityGroups defines that kubernetes manages the OpenStack security groups
	// for now, that means that we'll create two security groups, one allowing SSH
	// and API access from everywhere, and another one that allows all traffic to/from
	// machines belonging to that group. In the future, we could make this more flexible.
	ManagedSecurityGroups bool `json:"managedSecurityGroups"`

	// DisablePortSecurity disables the port security of the network created for the
	// Kubernetes cluster, which also disables SecurityGroups
	DisablePortSecurity bool `json:"disablePortSecurity,omitempty"`

	// Tags for all resources in cluster
	Tags []string `json:"tags,omitempty"`

	// Default: True. In case of server tag errors, set to False
	DisableServerTags bool `json:"disableServerTags,omitempty"`

	// CAKeyPair is the key pair for ca certs.
	CAKeyPair KeyPair `json:"caKeyPair,omitempty"`

	//EtcdCAKeyPair is the key pair for etcd.
	EtcdCAKeyPair KeyPair `json:"etcdCAKeyPair,omitempty"`

	// FrontProxyCAKeyPair is the key pair for FrontProxyKeyPair.
	FrontProxyCAKeyPair KeyPair `json:"frontProxyCAKeyPair,omitempty"`

	// SAKeyPair is the service account key pair.
	SAKeyPair KeyPair `json:"saKeyPair,omitempty"`

	// ClusterConfiguration holds the cluster-wide information used during a
	// kubeadm init call.
	ClusterConfiguration kubeadmv1beta1.ClusterConfiguration `json:"clusterConfiguration,omitempty"`
}

// KeyPair is how operators can supply custom keypairs for kubeadm to use.
type KeyPair struct {
	// base64 encoded cert and key
	Cert []byte `json:"cert,omitempty"`
	Key  []byte `json:"key,omitempty"`
}

// HasCertAndKey returns whether a keypair contains cert and key of non-zero length.
func (kp *KeyPair) HasCertAndKey() bool {
	return len(kp.Cert) != 0 && len(kp.Key) != 0
}

type ExternalRouterIPParam struct {
	// The FixedIP in the corresponding subnet
	FixedIP string `json:"fixedIP,omitempty"`
	// The subnet in which the FixedIP is used for the Gateway of this router
	Subnet SubnetParam `json:"subnet"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OpenstackClusterProviderStatus contains the status fields
// relevant to OpenStack in the cluster object.
// +k8s:openapi-gen=true
type OpenstackClusterProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Network contains all information about the created OpenStack Network.
	// It includes Subnets and Router.
	Network *Network `json:"network,omitempty"`

	// ControlPlaneSecurityGroups contains all the information about the OpenStack
	// Security Group that needs to be applied to control plane nodes.
	// TODO: Maybe instead of two properties, we add a property to the group?
	ControlPlaneSecurityGroup *SecurityGroup `json:"controlPlaneSecurityGroup,omitempty"`

	// GlobalSecurityGroup contains all the information about the OpenStack Security
	// Group that needs to be applied to all nodes, both control plane and worker nodes.
	GlobalSecurityGroup *SecurityGroup `json:"globalSecurityGroup,omitempty"`
}

// Network represents basic information about the associated OpenStach Neutron Network
type Network struct {
	Name string `json:"name"`
	ID   string `json:"id"`

	Subnet *Subnet `json:"subnet,omitempty"`
	Router *Router `json:"router,omitempty"`

	// Be careful when using APIServerLoadBalancer, because this field is optional and therefore not
	// set in all cases
	APIServerLoadBalancer *LoadBalancer `json:"apiServerLoadBalancer,omitempty"`
}

// Subnet represents basic information about the associated OpenStack Neutron Subnet
type Subnet struct {
	Name string `json:"name"`
	ID   string `json:"id"`

	CIDR string `json:"cidr"`
}

// Router represents basic information about the associated OpenStack Neutron Router
type Router struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// LoadBalancer represents basic information about the associated OpenStack LoadBalancer
type LoadBalancer struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	IP         string `json:"ip"`
	InternalIP string `json:"internalIP"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OpenstackClusterProviderStatus contains the status fields
// relevant to OpenStack in the cluster object.
// +k8s:openapi-gen=true
type OpenstackMachineProviderStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	//TODO: add provider status
}

func init() {
	SchemeBuilder.Register(&OpenstackProviderSpec{})
	SchemeBuilder.Register(&OpenstackClusterProviderSpec{})
	SchemeBuilder.Register(&OpenstackClusterProviderStatus{})
	SchemeBuilder.Register(&OpenstackMachineProviderStatus{})
}
