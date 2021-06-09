package install

const (
	AgentContainerName        = "traffic-agent"
	AgentAnnotationVolumeName = "traffic-annotations"
	AgentInjectorTLSName      = "agent-injector-tls"
	DomainPrefix              = "telepresence.getambassador.io/"
	InjectAnnotation          = DomainPrefix + "inject-" + AgentContainerName
	ManagerAppName            = "traffic-manager"
	ManagerPortHTTP           = 8081
	ManagerPortHTTPS          = 8443
	TelAppMountPoint          = "/tel_app_mounts"
)
