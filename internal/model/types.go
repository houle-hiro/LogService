package model

type FileInfo struct {
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	ModifiedTime string `json:"modified_time"`
}

type QueryRequest struct {
	Name   string `json:"name"`
	Commit string `json:"commit"`
}

type QueryResponse struct {
	Files []FileInfo `json:"files"`
	Count int        `json:"count"`
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
