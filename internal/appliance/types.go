package appliance

import "time"

type TLSField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
	Help        string `json:"help,omitempty"`
}

type TLSProvider struct {
	ID           string     `json:"id"`
	Label        string     `json:"label"`
	Description  string     `json:"description,omitempty"`
	Instructions []string   `json:"instructions,omitempty"`
	DocsURL      string     `json:"docsUrl,omitempty"`
	Fields       []TLSField `json:"fields,omitempty"`
}

type DynamicDNSProvider struct {
	ID           string     `json:"id"`
	Label        string     `json:"label"`
	Description  string     `json:"description,omitempty"`
	Instructions []string   `json:"instructions,omitempty"`
	DocsURL      string     `json:"docsUrl,omitempty"`
	Fields       []TLSField `json:"fields,omitempty"`
}

type ChallengeOption struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	Description  string   `json:"description,omitempty"`
	Instructions []string `json:"instructions,omitempty"`
	Recommended  bool     `json:"recommended,omitempty"`
}

type TLSConfig struct {
	Domain        string            `json:"domain"`
	Email         string            `json:"email"`
	ChallengeType string            `json:"challengeType"`
	Provider      string            `json:"provider,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	ConfiguredAt  time.Time         `json:"configuredAt,omitempty"`
	AppliedAt     time.Time         `json:"appliedAt,omitempty"`
}

type UpdateState struct {
	CurrentTag       string    `json:"currentTag"`
	PreviousTag      string    `json:"previousTag,omitempty"`
	CurrentAgentTag  string    `json:"currentAgentTag,omitempty"`
	PreviousAgentTag string    `json:"previousAgentTag,omitempty"`
	LastCheckedTag   string    `json:"lastCheckedTag,omitempty"`
	LastCheckedURL   string    `json:"lastCheckedURL,omitempty"`
	LastCheckedAt    time.Time `json:"lastCheckedAt,omitempty"`
	LastCheckError   string    `json:"lastCheckError,omitempty"`
	LastAction       string    `json:"lastAction,omitempty"`
	LastMessage      string    `json:"lastMessage,omitempty"`
	LastActionAt     time.Time `json:"lastActionAt,omitempty"`
}

type State struct {
	TLS        TLSConfig        `json:"tls"`
	DynamicDNS DynamicDNSConfig `json:"dynamicDns"`
	Update     UpdateState      `json:"update"`
}

type DynamicDNSConfig struct {
	Enabled         bool              `json:"enabled"`
	Provider        string            `json:"provider,omitempty"`
	Zone            string            `json:"zone,omitempty"`
	RecordName      string            `json:"recordName,omitempty"`
	IntervalMinutes int               `json:"intervalMinutes,omitempty"`
	Env             map[string]string `json:"env,omitempty"`
	LastKnownIP     string            `json:"lastKnownIp,omitempty"`
	LastCheckedAt   time.Time         `json:"lastCheckedAt,omitempty"`
	LastSyncedAt    time.Time         `json:"lastSyncedAt,omitempty"`
	LastError       string            `json:"lastError,omitempty"`
	ConfiguredAt    time.Time         `json:"configuredAt,omitempty"`
}

type StatusResponse struct {
	Service           string           `json:"service"`
	DockerManaged     bool             `json:"dockerManaged"`
	StackName         string           `json:"stackName"`
	StackFile         string           `json:"stackFile"`
	StackEnvFile      string           `json:"stackEnvFile"`
	CurrentTag        string           `json:"currentTag"`
	CurrentImage      string           `json:"currentImage"`
	CurrentAgentTag   string           `json:"currentAgentTag"`
	CurrentAgentImage string           `json:"currentAgentImage"`
	LastAction        string           `json:"lastAction,omitempty"`
	LastMessage       string           `json:"lastMessage,omitempty"`
	LastActionAt      time.Time        `json:"lastActionAt,omitempty"`
	TLS               TLSConfig        `json:"tls"`
	DynamicDNS        DynamicDNSConfig `json:"dynamicDns"`
	CurrentPublicIP   string           `json:"currentPublicIp,omitempty"`
}

type UpdateCheckResponse struct {
	CurrentTag      string    `json:"currentTag"`
	LatestTag       string    `json:"latestTag,omitempty"`
	UpdateAvailable bool      `json:"updateAvailable"`
	ReleaseURL      string    `json:"releaseUrl,omitempty"`
	CheckedAt       time.Time `json:"checkedAt"`
	Source          string    `json:"source"`
	Warning         string    `json:"warning,omitempty"`
}

type ApplyUpdateRequest struct {
	Tag          string `json:"tag"`
	IncludeAgent bool   `json:"includeAgent,omitempty"`
}

type ActionResponse struct {
	Status          string    `json:"status"`
	Message         string    `json:"message"`
	CurrentTag      string    `json:"currentTag,omitempty"`
	CurrentAgentTag string    `json:"currentAgentTag,omitempty"`
	AppliedAt       time.Time `json:"appliedAt,omitempty"`
}
