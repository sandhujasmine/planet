package constants

const (
	// KubectlConfigPath is the path to kubectl configuration file
	KubectlConfigPath = "/etc/kubernetes/kubectl.kubeconfig"
	// SchedulerConfigPath is the path to kube-scheduler configuration file
	SchedulerConfigPath = "/etc/kubernetes/scheduler.kubeconfig"
	// ProxyConfigPath is the path to kube-proxy configuration file
	ProxyConfigPath = "/etc/kubernetes/proxy.kubeconfig"
	// KubeletConfigPath is the path to kubelet configuration file
	KubeletConfigPath = "/etc/kubernetes/kubelet.kubeconfig"
	// CoreDNSConfigPath is the path to coredns configuration file
	CoreDNSConfigPath = "/etc/kubernetes/coredns.kubeconfig"
	// AgentConfigPath is the path to agent configuration file
	AgentConfigPath = "/etc/kubernetes/planet-agent.kubeconfig"

	// DNSResourceName specifies the name for the DNS resources
	DNSResourceName = "kube-dns"

	// CoreDNSConfigMapName is the location of the user supplied configmap for CoreDNS configuration
	CoreDNSConfigMapName = "coredns"

	// EnvironmentConfigMapName is the location of the user supplied configmap for cluster environment
	EnvironmentConfigMapName = "cluster-env"

	// ExitCodeUnknown is equivalent to EX_SOFTWARE as defined by sysexits(3)
	ExitCodeUnknown = 70

	// SharedReadMask is a file mask with read access for everyone
	SharedReadMask = 0644

	// SharedReadWriteMask is a mask for a shared file with read/write access for everyone
	SharedReadWriteMask = 0666

	// SharedDirMask is a mask for shared directories
	SharedDirMask = 0755

	// SystemdUnitPath specifies the path for user systemd units
	SystemdUnitPath = "/etc/systemd/system"

	// CloudProviderAWS defines the name of the AWS cloud provider used to
	// setup AWS integration in kubernetes
	CloudProviderAWS = "aws"
	// CloudProviderGCE is the Google Compute Engine cloud provider ID
	CloudProviderGCE = "gce"

	// OverlayInterfaceName is the name of the linux network interface connected to the overlay network
	OverlayInterfaceName = "docker0"

	// APIServerDNSName is the domain name of a current leader server
	APIServerDNSName = "leader.telekube.local"
	// TelekubeDomain is the domain for local telekube cluster
	TelekubeDomain = "telekube.local"
)

var (
	// GravityDataDir is the directory where gravity data is stored in planet
	GravityDataDir = "/var/lib/gravity"
)
