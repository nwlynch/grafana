// Code generated - EDITING IS FUTILE. DO NOT EDIT.

package v0alpha1

// +k8s:openapi-gen=true
type AlertStateEntry struct {
	Timestamp int64  `json:"timestamp"`
	Line      string `json:"line"`
}

// NewAlertStateEntry creates a new AlertStateEntry object.
func NewAlertStateEntry() *AlertStateEntry {
	return &AlertStateEntry{}
}

// +k8s:openapi-gen=true
type CreateAlertstatequery struct {
	Entries []AlertStateEntry `json:"entries"`
}

// NewCreateAlertstatequery creates a new CreateAlertstatequery object.
func NewCreateAlertstatequery() *CreateAlertstatequery {
	return &CreateAlertstatequery{
		Entries: []AlertStateEntry{},
	}
}
