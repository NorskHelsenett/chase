package types

type ServerPatch struct {
	URL                *string `json:"url"`
	Active             *bool   `json:"active"`
	FollowRedirect     *bool   `json:"follow_redirect"`
	AllowInsecure      *bool   `json:"allow_insecure"`
	ExpectedStatusCode *int    `json:"expected_status"`
	Comment            *string `json:"comment"`
	UpdateInterval     *int    `json:"update_interval"`
}
