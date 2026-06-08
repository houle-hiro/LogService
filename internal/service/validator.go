package service

import (
	"fmt"
	"regexp"
)

var filenamePattern = regexp.MustCompile(`^[a-zA-Z0-9\-]+_[a-zA-Z0-9\-]+_\d{14}\.zip$`)

func ValidateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("文件名为空")
	}
	
	if !filenamePattern.MatchString(filename) {
		return fmt.Errorf("文件名格式错误，正确格式: {名称}_{commit_id}_{datetime}.zip，datetime 为 14 位数字")
	}
	
	return nil
}

func BuildPrefix(name, commit string) string {
	if name != "" && commit != "" {
		return fmt.Sprintf("%s_%s_", name, commit)
	}
	if name != "" {
		return name + "_"
	}
	if commit != "" {
		return "_" + commit + "_"
	}
	return ""
}
