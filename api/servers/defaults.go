package servers

const defaultExpectedStatusCode = 200
const defaultUpdateIntervalMinutes = 15

type serverCreateRequest struct {
	URL                string  `json:"url" binding:"required"`
	Active             *bool   `json:"active"`
	FollowRedirect     *bool   `json:"follow_redirect"`
	AllowInsecure      *bool   `json:"allow_insecure"`
	ExpectedStatusCode *int    `json:"expected_status"`
	Comment            *string `json:"comment"`
	UpdateInterval     *int    `json:"update_interval"`
}

func defaultServerModel() Server {
	return Server{
		Active:             true,
		FollowRedirect:     true,
		AllowInsecure:      false,
		ExpectedStatusCode: defaultExpectedStatusCode,
		UpdateInterval:     defaultUpdateIntervalMinutes,
	}
}

func serverFromCreateRequest(request serverCreateRequest) Server {
	server := defaultServerModel()
	server.URL = request.URL

	if request.Active != nil {
		server.Active = *request.Active
	}
	if request.FollowRedirect != nil {
		server.FollowRedirect = *request.FollowRedirect
	}
	if request.AllowInsecure != nil {
		server.AllowInsecure = *request.AllowInsecure
	}
	if request.ExpectedStatusCode != nil {
		server.ExpectedStatusCode = *request.ExpectedStatusCode
	}
	if request.Comment != nil {
		server.Comment = *request.Comment
	}
	if request.UpdateInterval != nil && *request.UpdateInterval > 0 {
		server.UpdateInterval = *request.UpdateInterval
	}

	return server
}
