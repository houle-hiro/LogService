package model

// FileInfo 文件信息
type FileInfo struct {
	Username     string `json:"username,omitempty"` // 所属用户（ListAll 时填充）
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	ModifiedTime string `json:"modified_time"`
}

// QueryResponse 查询结果
type QueryResponse struct {
	Username string     `json:"username"`
	Keyword  string     `json:"keyword,omitempty"`
	Files    []FileInfo `json:"files"`
	Count    int        `json:"count"`
}

// UploadResponse 上传结果
type UploadResponse struct {
	Username    string `json:"username"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	Overwritten bool   `json:"overwritten"`
}

// SessionInfo 当前会话用户
type SessionInfo struct {
	Username string `json:"username"`
}

// Response 通用响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}