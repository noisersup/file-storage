package server

// RESPONSES

type ErrResponse struct {
	Error string `json:"error"`
}

type ListFilesResponse struct {
	Files []ListedFile `json:"files"`
	Error string       `json:"error"`
}

type ListedFile struct {
	Name        string `json:"name"`
	IsDirectory bool   `json:"isDirectory"`
}

// REQUESTS
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
