package service

import "testing"

func TestValidateFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "有效文件名",
			filename: "myapp_abc123_20240101120000.zip",
			wantErr:  false,
		},
		{
			name:     "另一个有效文件名",
			filename: "service_xyz789_20231231235959.zip",
			wantErr:  false,
		},
		{
			name:     "空文件名",
			filename: "",
			wantErr:  true,
		},
		{
			name:     "缺少扩展名",
			filename: "myapp_abc123_20240101120000",
			wantErr:  true,
		},
		{
			name:     "错误的扩展名",
			filename: "myapp_abc123_20240101120000.tar",
			wantErr:  true,
		},
		{
			name:     "缺少时间戳",
			filename: "myapp_abc123.zip",
			wantErr:  true,
		},
		{
			name:     "时间戳位数不足",
			filename: "myapp_abc123_2024010112000.zip",
			wantErr:  true,
		},
		{
			name:     "时间戳位数过多",
			filename: "myapp_abc123_202401011200000.zip",
			wantErr:  true,
		},
		{
			name:     "缺少下划线分隔",
			filename: "myapp-abc123-20240101120000.zip",
			wantErr:  true,
		},
		{
			name:     "包含非法字符",
			filename: "my app_abc123_20240101120000.zip",
			wantErr:  true,
		},
		{
			name:     "只有两部分",
			filename: "myapp_20240101120000.zip",
			wantErr:  true,
		},
		{
			name:     "空名称",
			filename: "_abc123_20240101120000.zip",
			wantErr:  true,
		},
		{
			name:     "空 commit_id",
			filename: "myapp__20240101120000.zip",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilename(%q) error = %v, wantErr %v",
					tt.filename, err, tt.wantErr)
			}
		})
	}
}

func TestBuildPrefix(t *testing.T) {
	tests := []struct {
		name   string
		commit string
		want   string
	}{
		{
			name:   "myapp",
			commit: "abc123",
			want:   "myapp_abc123_",
		},
		{
			name:   "myapp",
			commit: "",
			want:   "myapp_",
		},
		{
			name:   "",
			commit: "abc123",
			want:   "_abc123_",
		},
		{
			name:   "",
			commit: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+tt.commit, func(t *testing.T) {
			got := BuildPrefix(tt.name, tt.commit)
			if got != tt.want {
				t.Errorf("BuildPrefix(%q, %q) = %q, want %q",
					tt.name, tt.commit, got, tt.want)
			}
		})
	}
}
