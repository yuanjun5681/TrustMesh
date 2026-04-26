package protocol

// ── Inbound: ClawHire → TrustMesh ─────────────────────────────────────────

// ClawHireAgreedReward describes the compensation terms attached to a ClawHire task.
type ClawHireAgreedReward struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// ClawHireTaskAwardedPayload is the message body of clawhire.task.awarded.
// TrustMesh creates a planning task in the user's ClawHire project upon receipt.
type ClawHireTaskAwardedPayload struct {
	TaskID       string                `json:"taskId"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	Category     string                `json:"category,omitempty"`
	ContractID   string                `json:"contractId,omitempty"`
	AgreedReward *ClawHireAgreedReward `json:"agreedReward,omitempty"`
	Deadline     string                `json:"deadline,omitempty"`
	RequesterID  string                `json:"requesterId,omitempty"`
}

// ClawHireSubmissionAcceptedPayload is the message body of clawhire.submission.accepted.
// TrustMesh records the acceptance event on the corresponding task.
type ClawHireSubmissionAcceptedPayload struct {
	TaskID       string `json:"taskId"`
	SubmissionID string `json:"submissionId,omitempty"`
	ContractID   string `json:"contractId,omitempty"`
	AcceptedAt   string `json:"acceptedAt,omitempty"`
}

// ClawHireSubmissionRejectedPayload is the message body of clawhire.submission.rejected.
// TrustMesh records the rejection event; re-execution is up to the user / PM agent.
type ClawHireSubmissionRejectedPayload struct {
	TaskID       string `json:"taskId"`
	SubmissionID string `json:"submissionId,omitempty"`
	Reason       string `json:"reason"`
	RejectedAt   string `json:"rejectedAt,omitempty"`
}

// ── Outbound: TrustMesh → ClawHire ────────────────────────────────────────

// ClawHireTaskStartedPayload is the message body of clawhire.task.started.
// Sent when TrustMesh dispatches the first execution agent after plan is finalized.
type ClawHireTaskStartedPayload struct {
	TaskID     string `json:"taskId"`
	ContractID string `json:"contractId,omitempty"`
	StartedAt  string `json:"startedAt"`
}

// ClawHireSubmissionArtifact is one deliverable referenced by a submission.
type ClawHireSubmissionArtifact struct {
	Type string `json:"type"`
	URL  string `json:"url,omitempty"`
	Name string `json:"name,omitempty"`
}

// ClawHireSubmissionEvidence groups supporting evidence items for a submission.
type ClawHireSubmissionEvidence struct {
	Type  string   `json:"type"`
	Items []string `json:"items,omitempty"`
}

// ClawHireSubmissionCreatedPayload is the message body of clawhire.submission.created.
// Sent when a todo in an externally-linked task completes.
type ClawHireSubmissionCreatedPayload struct {
	TaskID      string                       `json:"taskId"`
	ContractID  string                       `json:"contractId,omitempty"`
	Summary     string                       `json:"summary"`
	Artifacts   []ClawHireSubmissionArtifact `json:"artifacts,omitempty"`
	Evidence    *ClawHireSubmissionEvidence  `json:"evidence,omitempty"`
	SubmittedAt string                       `json:"submittedAt"`
}

// ClawHireProgressReportedPayload is the message body of clawhire.progress.reported.
// Sent when a todo in an externally-linked task reports progress.
type ClawHireProgressReportedPayload struct {
	TaskID     string  `json:"taskId"`
	ContractID string  `json:"contractId,omitempty"`
	Stage      string  `json:"stage,omitempty"`
	Percent    float64 `json:"percent,omitempty"`
	Summary    string  `json:"summary"`
	ReportedAt string  `json:"reportedAt"`
}

// ClawHireConnectionEstablishedPayload is the message body of clawhire.connection.established.
// Sent by TrustMesh after a user successfully binds their ClawHire account.
type ClawHireConnectionEstablishedPayload struct {
	TrustMeshNodeID string `json:"trustMeshNodeId"`
	RemoteUserID    string `json:"remoteUserId"`
	LinkedAt        string `json:"linkedAt"`
}

// ClawHireConnectionRemovedPayload is the message body of clawhire.connection.removed.
// Sent by TrustMesh after a user unbinds their ClawHire account.
type ClawHireConnectionRemovedPayload struct {
	TrustMeshNodeID string `json:"trustMeshNodeId"`
	RemoteUserID    string `json:"remoteUserId"`
	RemovedAt       string `json:"removedAt"`
}
