package cfg

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	RestoreS3EndpointEnvName        = "RESTORE_S3ENDPOINT"
	RestoreS3AccessKeyIDEnvName     = "RESTORE_ACCESSKEYID"
	RestoreS3SecretAccessKeyEnvName = "RESTORE_SECRETACCESSKEY"
	ResticRepositoryEnvName         = "RESTIC_REPOSITORY"
	ResticPasswordEnvName           = "RESTIC_PASSWORD"
	AwsAccessKeyIDEnvName           = "AWS_ACCESS_KEY_ID"
	AwsSecretAccessKeyEnvName       = "AWS_SECRET_ACCESS_KEY"
)

// Configuration holds a strongly-typed tree of the configuration
type Configuration struct {
	MountPath                        string `koanf:"datapath"`
	BackupAnnotation                 string `koanf:"annotation"`
	BackupCommandAnnotation          string `koanf:"backupcommandannotation"`
	FileExtensionAnnotation          string `koanf:"fileextensionannotation"`
	ServiceAccount                   string `koanf:"podexecaccountname"`
	BackupCheckSchedule              string `koanf:"checkschedule"`
	GlobalAccessKey                  string `koanf:"globalaccesskeyid"`
	GlobalKeepJobs                   int    `koanf:"globalkeepjobs"`
	GlobalRepoPassword               string `koanf:"globalrepopassword"`
	GlobalRestoreS3AccessKey         string `koanf:"globalrestores3accesskeyid"`
	GlobalRestoreS3Bucket            string `koanf:"globalrestores3bucket"`
	GlobalRestoreS3Endpoint          string `koanf:"globalrestores3endpoint"`
	GlobalRestoreS3SecretAccessKey   string `koanf:"globalrestores3secretaccesskey"`
	GlobalS3Bucket                   string `koanf:"globals3bucket"`
	GlobalS3Endpoint                 string `koanf:"globals3endpoint"`
	GlobalSecretAccessKey            string `koanf:"globalsecretaccesskey"`
	GlobalStatsURL                   string `koanf:"globalstatsurl"`
	GlobalConcurrentArchiveJobsLimit int    `koanf:"global-concurrent-archive-jobs-limit"`
	GlobalConcurrentBackupJobsLimit  int    `koanf:"global-concurrent-backup-jobs-limit"`
	GlobalConcurrentCheckJobsLimit   int    `koanf:"global-concurrent-check-jobs-limit"`
	GlobalConcurrentPruneJobsLimit   int    `koanf:"global-concurrent-prune-jobs-limit"`
	GlobalConcurrentRestoreJobsLimit int    `koanf:"global-concurrent-restore-jobs-limit"`
	GlobalCPUResourceRequest         string `koanf:"global-cpu-request"`
	GlobalCPUResourceLimit           string `koanf:"global-cpu-limit"`
	GlobalMemoryResourceRequest      string `koanf:"global-memory-request"`
	GlobalMemoryResourceLimit        string `koanf:"global-memory-limit"`
	BackupImage                      string `koanf:"image"`
	MetricsBindAddress               string `koanf:"metrics-bindaddress"`
	PodExecRoleName                  string `koanf:"podexecrolename"`
	PodFilter                        string `koanf:"podfilter"`
	PromURL                          string `koanf:"promurl"`
	RestartPolicy                    string `koanf:"restartpolicy"`

	// Enabling this will ensure there is only one active controller manager.
	EnableLeaderElection bool   `koanf:"enable-leader-election"`
	LogLevel             string `koanf:"log-level"`
	OperatorNamespace    string `koanf:"operator-namespace"`
}

var (
	Config = NewDefaultConfig()
)

// NewDefaultConfig retrieves the config with sane defaults
func NewDefaultConfig() *Configuration {
	return &Configuration{
		MountPath:               "/data",
		BackupAnnotation:        "k8up.syn.tools/backup",
		BackupCommandAnnotation: "k8up.syn.tools/backupcommand",
		FileExtensionAnnotation: "k8up.syn.tools/file-extension",
		ServiceAccount:          "pod-executor",
		BackupCheckSchedule:     "0 0 * * 0",
		GlobalKeepJobs:          6,
		BackupImage:             "quay.io/vshn/wrestic:latest",
		PodExecRoleName:         "pod-executor",
		RestartPolicy:           "OnFailure",
		MetricsBindAddress:      ":8080",
		PodFilter:               "backupPod=true",
		EnableLeaderElection:    true,
		OperatorNamespace:       "k8up-system",
	}
}

func (c Configuration) ValidateSyntax() error {
	if _, err := resource.ParseQuantity(c.GlobalMemoryResourceRequest); err != nil && c.GlobalMemoryResourceRequest != "" {
		return fmt.Errorf("cannot parse global memory request: %v", err)
	}
	if _, err := resource.ParseQuantity(c.GlobalMemoryResourceLimit); err != nil && c.GlobalMemoryResourceLimit != "" {
		return fmt.Errorf("cannot parse global memory limit: %v", err)
	}
	if _, err := resource.ParseQuantity(c.GlobalCPUResourceRequest); err != nil && c.GlobalCPUResourceRequest != "" {
		return fmt.Errorf("cannot parse global CPU request: %v", err)
	}
	if _, err := resource.ParseQuantity(c.GlobalCPUResourceLimit); err != nil && c.GlobalCPUResourceLimit != "" {
		return fmt.Errorf("cannot parse global CPU limit: %v", err)
	}
	if c.OperatorNamespace == "" {
		return fmt.Errorf("operator namespace cannot be empty")
	}
	return nil
}

func (c Configuration) GetGlobalDefaultResources() (res corev1.ResourceRequirements) {
	if r, err := resource.ParseQuantity(c.GlobalMemoryResourceRequest); err == nil && c.GlobalMemoryResourceRequest != "" {
		if res.Requests == nil {
			res.Requests = make(corev1.ResourceList)
		}
		res.Requests[corev1.ResourceMemory] = r
	}
	if r, err := resource.ParseQuantity(c.GlobalCPUResourceRequest); err == nil && c.GlobalCPUResourceRequest != "" {
		if res.Requests == nil {
			res.Requests = make(corev1.ResourceList)
		}
		res.Requests[corev1.ResourceCPU] = r
	}
	if r, err := resource.ParseQuantity(c.GlobalMemoryResourceLimit); err == nil && c.GlobalMemoryResourceLimit != "" {
		if res.Limits == nil {
			res.Limits = make(corev1.ResourceList)
		}
		res.Limits[corev1.ResourceMemory] = r
	}
	if r, err := resource.ParseQuantity(c.GlobalCPUResourceLimit); err == nil && c.GlobalCPUResourceLimit != "" {
		if res.Limits == nil {
			res.Limits = make(corev1.ResourceList)
		}
		res.Limits[corev1.ResourceCPU] = r
	}
	return res
}

// GetGlobalRepository is a shortcut for building an S3 string "s3:<endpoint>/<bucket>"
func GetGlobalRepository() string {
	return fmt.Sprintf("s3:%s/%s", Config.GlobalS3Endpoint, Config.GlobalS3Bucket)
}
