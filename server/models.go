package server

// RESPONSES

type ErrResponse struct {
	Error string `json:"error"`
}

type ListFilesResponse struct {
	Files []string `json:"files"`
	Error string   `json:"error"`
}
