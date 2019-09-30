package model

type FileType string

func (f FileType) String() (ft string) {
	switch f {
	case FileTypeImage, FileTypeText, FileTypeVideo:
		ft = string(f)
	default:
		ft = string(FileTypeUnknown)
	}
	return ft
}

const (
	FileTypeUnknown FileType = "unknown"
	FileTypeImage            = "image"
	FileTypeVideo            = "video"
	FileTypeText             = "text"
)
